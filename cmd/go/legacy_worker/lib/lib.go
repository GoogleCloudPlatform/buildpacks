// Copyright 2025 Google LLC
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
package lib

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
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
	//go:embed converter/worker/gomod.tmpl
	goModTmplFile string
)

type fnInfo struct {
	Source  string
	Target  string
	Package string
}

// DetectFn detects if this is a Go 1.11 legacy worker function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !golang.IsGo111Runtime() {
		return gcp.OptOut("Only compatible with go111"), nil
	}
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

// BuildFn converts the function into an application and sets up the execution environment.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer(layerName, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}
	if err := ctx.SetFunctionsEnvVars(l); err != nil {
		return err
	}
	ctx.AddWebProcess([]string{golang.OutBin})

	fnTarget := os.Getenv(env.FunctionTarget)

	// Move the function source code into a subdirectory in order to construct the app in the main application root.
	if err := ctx.RemoveAll(fnSourceDir); err != nil {
		return err
	}
	if err := ctx.MkdirAll(fnSourceDir, 0755); err != nil {
		return err
	}
	// mindepth=1 excludes '.', '+' collects all file names before running the command.
	// Exclude serverless_function_source_code and .google* dir e.g. .googlebuild, .googleconfig
	command := fmt.Sprintf("find . -mindepth 1 -not -name %[1]s -prune -not -name %[2]q -prune -exec mv -t %[1]s {} +", fnSourceDir, ".google*")
	if _, err := ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserTimingAttribution); err != nil {
		return err
	}

	fnSource := filepath.Join(ctx.ApplicationRoot(), fnSourceDir)
	pkgName, err := extractPackageNameInDir(ctx, fnSource)
	if err != nil {
		return fmt.Errorf("extracting package name: %w", err)
	}
	fn := fnInfo{
		Source:  fnSource,
		Target:  fnTarget,
		Package: pkgName,
	}

	l.LaunchEnvironment.Default("X_GOOGLE_ENTRY_POINT", os.Getenv(env.FunctionTarget))
	triggerType := os.Getenv(env.FunctionSignatureType)
	if triggerType == "http" || triggerType == "" {
		triggerType = "HTTP_TRIGGER"
	}
	l.LaunchEnvironment.Default("X_GOOGLE_FUNCTION_TRIGGER_TYPE", triggerType)

	goMod := filepath.Join(fn.Source, "go.mod")
	goModExists, err := ctx.FileExists(goMod)
	if err != nil {
		return err
	}
	if !goModExists {
		return createMainVendored(ctx, fn)
	}
	isWriteable, err := ctx.IsWritable(goMod)
	if err != nil {
		return err
	}
	if !isWriteable {
		// Preempt an obscure failure mode: if go.mod is not writable then `go list -m` can fail saying:
		//     go: updates to go.sum needed, disabled by -mod=readonly
		return gcp.UserErrorf("go.mod exists but is not writable")
	}
	return createMainGoMod(ctx, fn)
}

/*
createMainGoMod creates the `main.go` and `go.mod` required to form a
module-based Go application that wraps the user function into a server.
The main application's Go module depends on the user function's Go module,
which is assumed to be a subdirectory of the main application's source code.

ctx.ApplicationRoot()
├── go.mod // `module functions.local/app`
├── main.go
└── serverless_function_source_code // assumed to aleady exist

	├── go.mod // `module <user's module name>`
	├── fn.go
	└── ...
*/
func createMainGoMod(ctx *gcp.Context, fn fnInfo) error {
	l, err := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", gopathLayerName, err)
	}
	l.BuildEnvironment.Override("GOPATH", l.Path)
	if err := ctx.Setenv("GOPATH", l.Path); err != nil {
		return err
	}

	fnMod, fnPackage, err := moduleAndPackageNames(ctx, fn)
	if err != nil {
		return fmt.Errorf("extracting module and package names: %w", err)
	}
	fn.Package = fnPackage

	if err := createMainGoModFile(ctx, fnMod, filepath.Join(ctx.ApplicationRoot(), "go.mod")); err != nil {
		return fmt.Errorf("error creating `go.mod` for function application: %w", err)
	}

	return createMainGoFile(ctx, fn, filepath.Join(ctx.ApplicationRoot(), "main.go"))
}

func createMainGoModFile(ctx *gcp.Context, fnMod string, goModPath string) error {
	f, err := ctx.CreateFile(goModPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("worker_gomod").Parse(goModTmplFile)
	if err != nil {
		return err
	}

	tmplSubs := struct {
		AppModule string
		FnModule  string
		FnSource  string
	}{
		AppModule: appModule,
		FnModule:  fnMod,
		FnSource:  fnSourceDir,
	}

	return tmpl.Execute(f, tmplSubs)
}

// moduleAndPackageNames extracts the module name and package name of the function.
func moduleAndPackageNames(ctx *gcp.Context, fn fnInfo) (string, string, error) {
	result, err := ctx.Exec([]string{"go", "list", "-m"}, gcp.WithWorkDir(fn.Source), gcp.WithUserAttribution)
	if err != nil {
		return "", "", err
	}
	fnMod := result.Stdout
	// Add the module name to the the package name, such that go build will be able to find it,
	// if a directory with the package name is not at the app root. Otherwise, assume the package is at the module root.
	fnPackage := fnMod
	fnPackageExists, err := ctx.FileExists(ctx.ApplicationRoot(), fn.Package)
	if err != nil {
		return "", "", err
	}
	if fnPackageExists {
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
	l, err := ctx.Layer(gopathLayerName, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", gopathLayerName, err)
	}
	gopath := ctx.ApplicationRoot()
	gopathSrc := filepath.Join(gopath, "src")
	if err := ctx.MkdirAll(gopathSrc, 0755); err != nil {
		return err
	}
	l.BuildEnvironment.Override(env.Buildable, appModule+"/main")
	l.BuildEnvironment.Override("GOPATH", gopath)
	l.BuildEnvironment.Override("GO111MODULE", "auto")
	if err := ctx.Setenv("GOPATH", gopath); err != nil {
		return err
	}

	appPath := filepath.Join(gopathSrc, appModule, "main")
	if err := ctx.MkdirAll(appPath, 0755); err != nil {
		return err
	}

	// We move the function source (including any vendored deps) into GOPATH.
	if err := ctx.Rename(fn.Source, filepath.Join(gopathSrc, fn.Package)); err != nil {
		return err
	}
	return createMainGoFile(ctx, fn, filepath.Join(appPath, "main.go"))
}

func createMainGoFile(ctx *gcp.Context, fn fnInfo, main string) error {
	f, err := ctx.CreateFile(main)
	if err != nil {
		return err
	}
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
func extractPackageNameInDir(ctx *gcp.Context, source string) (string, error) {
	script := filepath.Join(ctx.BuildpackRoot(), "converter", "get_package", "main.go")
	cacheDir, err := ctx.TempDir("app")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %w", err)
	}
	result, err := ctx.Exec([]string{"go", "run", script, "-dir", source}, gcp.WithEnv("GOCACHE="+cacheDir), gcp.WithUserAttribution)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
