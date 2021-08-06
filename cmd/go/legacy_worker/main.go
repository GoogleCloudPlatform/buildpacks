// Copyright 2021 Google LLC
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

// Implements the legacy GCF Go 1.11 worker buildpack.
// The legacy_worker buildpack converts a function into an application and sets up the execution environment.
package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

const (
	layerName       = "legacy-worker"
	gopathLayerName = "gopath"
	appModule       = "functions.local/app"
	fnSourceDir     = "serverless_function_source_code"
)

var (
	googleDirs = []string{fnSourceDir, ".googlebuild", ".googleconfig"}
	//go:embed converter/worker/main.tmpl
	workerTmplFile string
)

type fnInfo struct {
	Source  string
	Target  string
	Package string
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName, gcp.LaunchLayer)
	ctx.SetFunctionsEnvVars(l)
	ctx.AddWebProcess([]string{golang.OutBin})

	fnTarget := os.Getenv(env.FunctionTarget)

	// Move the function source code into a subdirectory in order to construct the app in the main application root.
	ctx.RemoveAll(fnSourceDir)
	ctx.MkdirAll(fnSourceDir, 0755)
	// mindepth=1 excludes '.', '+' collects all file names before running the command.
	// Exclude serverless_function_source_code and .google* dir e.g. .googlebuild, .googleconfig
	command := fmt.Sprintf("find . -mindepth 1 -not -name %[1]s -prune -not -name %[2]q -prune -exec mv -t %[1]s {} +", fnSourceDir, ".google*")
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserTimingAttribution)

	fnSource := filepath.Join(ctx.ApplicationRoot(), fnSourceDir)
	fn := fnInfo{
		Source:  fnSource,
		Target:  fnTarget,
		Package: extractPackageNameInDir(ctx, fnSource),
	}

	goMod := filepath.Join(fn.Source, "go.mod")
	if !ctx.FileExists(goMod) {
		return createMainVendored(ctx, fn)
	} else if !ctx.IsWritable(goMod) {
		// Preempt an obscure failure mode: if go.mod is not writable then `go list -m` can fail saying:
		//     go: updates to go.sum needed, disabled by -mod=readonly
		return gcp.UserErrorf("go.mod exists but is not writable")
	}
	if ctx.FileExists(fn.Source, "vendor") {
		return createMainGoModVendored(ctx, fn)
	}

	l.LaunchEnvironment.Default("X_GOOGLE_ENTRY_POINT", os.Getenv(env.FunctionTarget))
	triggerType := os.Getenv(env.FunctionSignatureType)
	if triggerType == "http" || triggerType == "" {
		triggerType = "HTTP_TRIGGER"
	}
	l.LaunchEnvironment.Default("X_GOOGLE_FUNCTION_TRIGGER_TYPE", triggerType)

	return createMainGoMod(ctx, fn)
}

func createMainGoMod(ctx *gcp.Context, fn fnInfo) error {
	l := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	l.BuildEnvironment.Override("GOPATH", l.Path)
	ctx.Setenv("GOPATH", l.Path)

	// If the function source does not include a go.sum, `go list` will fail under Go 1.16+.
	if !ctx.FileExists(fn.Source, "go.sum") {
		ctx.Logf(`go.sum not found, generating using "go mod tidy"`)
		golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "tidy"}, gcp.WithWorkDir(fn.Source), gcp.WithUserAttribution)
	}

	fnMod, fnPackage, err := moduleAndPackageNames(ctx, fn)
	if err != nil {
		return fmt.Errorf("extracting module and package names: %w", err)
	}
	fn.Package = fnPackage

	ctx.Exec([]string{"go", "mod", "init", appModule})
	ctx.Exec([]string{"go", "mod", "edit", "-require", fmt.Sprintf("%s@v0.0.0", fnMod)})
	ctx.Exec([]string{"go", "mod", "edit", "-replace", fmt.Sprintf("%s@v0.0.0=%s", fnMod, fn.Source)})

	return createMainGoFile(ctx, fn, filepath.Join(ctx.ApplicationRoot(), "main.go"))
}

func createMainGoModVendored(ctx *gcp.Context, fn fnInfo) error {
	l := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	l.BuildEnvironment.Override("GOPATH", l.Path)
	ctx.Setenv("GOPATH", l.Path)

	fnMod, fnPackage, err := moduleAndPackageNames(ctx, fn)
	if err != nil {
		return fmt.Errorf("extracting module and package names: %w", err)
	}
	fn.Package = fnPackage

	appVendorDir := filepath.Join(fn.Source, "vendor", appModule)
	ctx.MkdirAll(appVendorDir, 0755)
	ctx.Exec([]string{"go", "mod", "init", appModule}, gcp.WithWorkDir(appVendorDir))
	ctx.Exec([]string{"go", "mod", "edit", "-require", fmt.Sprintf("%s@v0.0.0", fnMod)}, gcp.WithWorkDir(appVendorDir))

	l.BuildEnvironment.Override(env.Buildable, appModule)
	l.BuildEnvironment.Override(golang.BuildDirEnv, fn.Source)

	return createMainGoFile(ctx, fn, filepath.Join(appVendorDir, "main.go"))
}

// moduleAndPackageNames extracts the module name and package name of the function.
func moduleAndPackageNames(ctx *gcp.Context, fn fnInfo) (string, string, error) {
	fnMod := ctx.Exec([]string{"go", "list", "-m"}, gcp.WithWorkDir(fn.Source), gcp.WithUserAttribution).Stdout
	// golang.org/ref/mod requires that package names in a replace contains at least one dot.
	if parts := strings.Split(fnMod, "/"); len(parts) > 0 && !strings.Contains(parts[0], ".") {
		return "", "", gcp.UserErrorf("the module path in the function's go.mod must contain a dot in the first path element before a slash, e.g. example.com/module, found: %s", fnMod)
	}
	// Add the module name to the the package name, such that go build will be able to find it,
	// if a directory with the package name is not at the app root. Otherwise, assume the package is at the module root.
	fnPackage := fnMod
	if ctx.FileExists(ctx.ApplicationRoot(), fn.Package) {
		fnPackage = fmt.Sprintf("%s/%s", fnMod, fn.Package)
	}
	return fnMod, fnPackage, nil
}

// createMainVendored creates the main.go file for vendored functions.
// This should only be run for Go 1.11 and 1.13.
// Go 1.11 and 1.13 on GCF allow for vendored go.mod deployments without a go.mod file.
// Note that despite the lack of a go.mod file, this does *not* mean that these are GOPATH deployments.
// These deployments were created by running `go mod vendor` and then .gcloudignoring the go.mod file,
// so that Go versions that don't natively handle gomod vendoring would be able to pick up the vendored deps.
// n.b. later versions of Go (1.14+) handle vendored go.mod files natively, and so we just use the go.mod route there.
func createMainVendored(ctx *gcp.Context, fn fnInfo) error {
	l := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	gopath := ctx.ApplicationRoot()
	gopathSrc := filepath.Join(gopath, "src")
	ctx.MkdirAll(gopathSrc, 0755)
	l.BuildEnvironment.Override(env.Buildable, appModule+"/main")
	l.BuildEnvironment.Override("GOPATH", gopath)
	l.BuildEnvironment.Override("GO111MODULE", "auto")
	ctx.Setenv("GOPATH", gopath)

	appPath := filepath.Join(gopathSrc, appModule, "main")
	ctx.MkdirAll(appPath, 0755)

	// We move the function source (including any vendored deps) into GOPATH.
	ctx.Rename(fn.Source, filepath.Join(gopathSrc, fn.Package))

	fnVendoredPath := filepath.Join(gopathSrc, fn.Package, "vendor")
	if ctx.FileExists(fnVendoredPath) {
		ctx.Exec([]string{"mv", fnVendoredPath, appPath}, gcp.WithUserTimingAttribution)
	}

	return createMainGoFile(ctx, fn, filepath.Join(appPath, "main.go"))
}

func createMainGoFile(ctx *gcp.Context, fn fnInfo, main string) error {
	f := ctx.CreateFile(main)
	defer f.Close()

	tmpl, err := template.New("worker_main").Parse(workerTmplFile)
	if err != nil {
		return err
	}

	return tmpl.Execute(f, fn)
}

// extractPackageNameInDir builds the script that does the extraction, and then runs it with the
// specified source directory.
// The parser is dependent on the language version being used, and it's highly likely that the buildpack binary
// will be built with a different version of the language than the function deployment. Building this script ensures
// that the version of Go used to build the function app will be the same as the version used to parse it.
func extractPackageNameInDir(ctx *gcp.Context, source string) string {
	script := filepath.Join(ctx.BuildpackRoot(), "converter", "get_package", "main.go")
	cacheDir := ctx.TempDir("app")
	return ctx.Exec([]string{"go", "run", script, "-dir", source}, gcp.WithEnv("GOCACHE="+cacheDir), gcp.WithUserAttribution).Stdout
}
