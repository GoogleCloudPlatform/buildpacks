// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implements go/functions_framework buildpack.
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName                 = "functions-framework"
	functionsFrameworkModule  = "github.com/GoogleCloudPlatform/functions-framework-go"
	functionsFrameworkPackage = functionsFrameworkModule + "/funcframework"
	functionsFrameworkVersion = "v1.0.1"
	appName                   = "serverless_function_app"
	fnSourceDir               = "serverless_function_source_code"
)

var (
	tmpl = template.Must(template.New("main").Parse(mainTextTemplate))
)

type fnInfo struct {
	Source  string
	Target  string
	Package string
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		ctx.OptIn("%s set", env.FunctionTarget)
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName)
	ctx.Setenv("GOPATH", l.Root)

	ctx.SetFunctionsEnvVars(l)

	fnTarget := os.Getenv(env.FunctionTarget)
	// TODO(b/154846199): For compatibility with GCF; this will be removed later.
	if fnTarget == "" {
		fnTarget = os.Getenv(env.FunctionTargetLaunch)
	}

	// Move the function source code into a subdirectory in order to construct the app in the main application root.
	ctx.RemoveAll(fnSourceDir)
	ctx.MkdirAll(fnSourceDir, 0755)
	// mindepth=1 excludes '.', '+' collects all file names before running the command.
	command := fmt.Sprintf("find . -mindepth 1 -not -name %[1]s -prune -exec mv -t %[1]s {} +", fnSourceDir)
	ctx.Exec([]string{"bash", "-c", command})

	fnSource := filepath.Join(ctx.ApplicationRoot(), fnSourceDir)
	fn := fnInfo{
		Source:  fnSource,
		Target:  fnTarget,
		Package: extractPackageNameInDir(ctx, fnSource),
	}

	if !ctx.FileExists(fn.Source, "go.mod") {
		// We require a go.mod file in all versions 1.14+.
		if !golang.SupportsNoGoMod(ctx) {
			return gcp.UserErrorf("function build requires go.mod file")
		}
		if err := createMainVendored(ctx, l, fn); err != nil {
			return err
		}
	} else {
		if err := createMainGoMod(ctx, fn); err != nil {
			return err
		}
	}

	ctx.AddWebProcess([]string{golang.OutBin})
	return nil
}

func createMainGoMod(ctx *gcp.Context, fn fnInfo) error {
	ctx.Exec([]string{"go", "mod", "init", appName})

	fnMod := ctx.ExecWithParams(gcp.ExecParams{
		Cmd: []string{"go", "list", "-m"},
		Dir: fn.Source,
	}).Stdout

	// Add the module name to the the package name, such that go build will be able to find it,
	// if a directory with the package name is not at the app root. Otherwise, assume the package is at the module root.
	if ctx.FileExists(ctx.ApplicationRoot(), fn.Package) {
		fn.Package = fmt.Sprintf("%s/%s", fnMod, fn.Package)
	} else {
		fn.Package = fnMod
	}

	ctx.Exec([]string{"go", "mod", "edit", "-require", fmt.Sprintf("%s@v0.0.0", fnMod)})
	ctx.Exec([]string{"go", "mod", "edit", "-replace", fmt.Sprintf("%s@v0.0.0=%s", fnMod, fn.Source)})

	// If the framework is not present in the function's go.mod, we require the current version.
	if specified, err := frameworkSpecified(ctx, fn.Source); err != nil {
		return fmt.Errorf("checking for functions framework dependency in go.mod: %w", err)
	} else if !specified {
		ctx.ExecUser([]string{"go", "get", fmt.Sprintf("%s@%s", functionsFrameworkModule, functionsFrameworkVersion)})
	}

	return createMainGoFile(ctx, fn, filepath.Join(ctx.ApplicationRoot(), "main.go"))
}

// createMainVendored creates the main.go file for vendored functions.
// This should only be run for Go 1.11 and 1.13.
// Go 1.11 and 1.13 on GCF allow for vendored go.mod deployments without a go.mod file.
// Note that despite the lack of a go.mod file, this does *not* mean that these are GOPATH deployments.
// These deployments were created by running `go mod vendor` and then .gcloudignoring the go.mod file,
// so that Go versions that don't natively handle gomod vendoring would be able to pick up the vendored deps.
// n.b. later versions of Go (1.14+) handle vendored go.mod files natively, and so we just use the go.mod route there.
func createMainVendored(ctx *gcp.Context, l *layers.Layer, fn fnInfo) error {
	ctx.OverrideBuildEnv(l, "GOPATH", ctx.ApplicationRoot())
	gopath := filepath.Join(ctx.ApplicationRoot(), "src")
	ctx.MkdirAll(gopath, 0755)

	ctx.OverrideBuildEnv(l, env.Buildable, appName+"/main")
	ctx.WriteMetadata(l, nil, layers.Build)

	appPath := filepath.Join(gopath, appName, "main")
	ctx.MkdirAll(appPath, 0755)

	// We move the function source (including any vendored deps) into the app's vendor directory, so that GOPATH can find it.
	ctx.Rename(fn.Source, filepath.Join(gopath, fn.Package))

	fnVendoredPath := filepath.Join(gopath, fn.Package, "vendor")

	fnFrameworkVendoredPath := filepath.Join(fnVendoredPath, functionsFrameworkPackage)
	if ctx.FileExists(fnFrameworkVendoredPath) {
		ctx.Exec([]string{"cp", "-r", fnVendoredPath, appPath})
	} else {
		// If the framework isn't in the user-provided vendor directory, we need to fetch it ourselves.
		// Create a temporary GOCACHE directory so GOPATH go get works.
		cache := ctx.TempDir("", appName)
		defer ctx.RemoveAll(cache)

		// The gopath version of `go get` doesn't allow tags, but does checkout the whole repo so we
		// can checkout the appropriate tag ourselves.
		ctx.ExecUserWithParams(gcp.ExecParams{
			Cmd: []string{"go", "get", functionsFrameworkPackage},
			Env: []string{"GOPATH=" + ctx.ApplicationRoot(), "GOCACHE=" + cache},
		}, gcp.UserErrorKeepStderrTail)
		ctx.ExecUserWithParams(gcp.ExecParams{
			Cmd: []string{"git", "checkout", functionsFrameworkVersion},
			Dir: filepath.Join(gopath, functionsFrameworkModule),
		}, gcp.UserErrorKeepStderrTail)
	}

	return createMainGoFile(ctx, fn, filepath.Join(appPath, "main.go"))
}

func createMainGoFile(ctx *gcp.Context, fn fnInfo, main string) error {
	f := ctx.CreateFile(main)
	defer f.Close()

	if err := tmpl.Execute(f, fn); err != nil {
		return fmt.Errorf("executing template: %v", err)
	}
	return nil
}

func frameworkSpecified(ctx *gcp.Context, fnSource string) (bool, error) {
	res, err := ctx.ExecWithErrWithParams(gcp.ExecParams{
		Cmd: []string{"go", "list", "-m", functionsFrameworkModule},
		Dir: fnSource,
	})
	if err == nil {
		return true, nil
	}
	if res != nil && strings.Contains(res.Stderr, "not a known dependency") {
		return false, nil
	}
	return false, err
}

// extractPackageNameInDir builds the script that does the extraction, and then runs it with the
// specified source directory.
// The parser is dependent on the language version being used, and it's highly likely that the buildpack binary
// will be built with a different version of the language than the function deployment. Building this script ensures
// that the version of Go used to build the function app will be the same as the version used to parse it.
func extractPackageNameInDir(ctx *gcp.Context, source string) string {
	scriptDir := filepath.Join(ctx.BuildpackRoot(), "converter", "get_package")
	cacheDir := ctx.TempDir("", appName)
	defer ctx.RemoveAll(cacheDir)
	return ctx.ExecUserWithParams(gcp.ExecParams{
		Cmd: []string{"go", "run", "main", "-dir", source},
		Env: []string{"GOPATH=" + scriptDir, "GOCACHE=" + cacheDir},
		Dir: scriptDir,
	}, gcp.UserErrorKeepStderrTail).Stdout
}
