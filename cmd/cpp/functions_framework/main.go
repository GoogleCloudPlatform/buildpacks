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

// Implements cpp/functions_framework buildpack.
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	mainLayerName               = "main"
	buildLayerName              = "build"
	vcpkgCacheLayerName         = "vcpkg-binary-cache"
	vcpkgLayerName              = "vcpkg"
	vcpkgVersion                = "9b9a6680b25872989c8eb0303d670f32e5cfe6a4"
	vcpkgToolVersion            = "2021-02-24-d67989bce1043b98092ac45996a8230a059a2d7e"
	vcpkgVersionPrefix          = "Vcpkg package management program version "
	vcpkgTripletName            = "x64-linux-nodebug"
	installLayerName            = "cpp"
	functionsFrameworkNamespace = "::google::cloud::functions"
)

type signatureInfo struct {
	ReturnType   string
	ArgumentType string
	WrapperType  string
}

var (
	vcpkgURL      = fmt.Sprintf("https://github.com/Microsoft/vcpkg/archive/%s.tar.gz", vcpkgVersion)
	mainTmpl      = template.Must(template.New("mainV0").Parse(mainTextTemplateV0))
	httpSignature = signatureInfo{
		ReturnType:   functionsFrameworkNamespace + "::HttpResponse",
		ArgumentType: functionsFrameworkNamespace + "::HttpRequest",
		WrapperType:  functionsFrameworkNamespace + "::UserHttpFunction",
	}
	cloudEventSignature = signatureInfo{
		ReturnType:   "void",
		ArgumentType: functionsFrameworkNamespace + "::CloudEvent",
		WrapperType:  functionsFrameworkNamespace + "::UserCloudEventFunction",
	}
)

type fnInfo struct {
	Target    string
	Namespace string
	ShortName string
	Signature signatureInfo
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func hasCppCode(ctx *gcp.Context) bool {
	if ctx.FileExists("CMakeLists.txt") {
		return true
	}
	if ctx.HasAtLeastOne("*.cc") {
		return true
	}
	if ctx.HasAtLeastOne("*.cxx") {
		return true
	}
	if ctx.HasAtLeastOne("*.cpp") {
		return true
	}
	return false
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !hasCppCode(ctx) {
		return gcp.OptOut("no C++ sources, nor a CMakeLists.txt file found"), nil
	}
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	vcpkgPath, err := installVcpkg(ctx)
	if err != nil {
		return err
	}

	vcpkgCache := ctx.Layer(vcpkgCacheLayerName, gcp.BuildLayer, gcp.CacheLayer)

	mainLayer := ctx.Layer(mainLayerName)
	ctx.SetFunctionsEnvVars(mainLayer)

	buildLayer := ctx.Layer(buildLayerName, gcp.BuildLayer, gcp.CacheLayer)

	fn := extractFnInfo(ctx)
	if err := createMainCppFile(ctx, fn, filepath.Join(mainLayer.Path, "main.cc")); err != nil {
		return err
	}

	ctx.Exec([]string{"cp", filepath.Join(ctx.BuildpackRoot(), "converter", "CMakeLists.txt"), filepath.Join(mainLayer.Path, "CMakeLists.txt")})

	vcpkgJSONDestinationFilename := filepath.Join(mainLayer.Path, "vcpkg.json")
	vcpkgJSONSourceFilename := filepath.Join(ctx.ApplicationRoot(), "vcpkg.json")

	if !ctx.FileExists(vcpkgJSONSourceFilename) {
		vcpkgJSONSourceFilename = filepath.Join(ctx.BuildpackRoot(), "converter", "vcpkg.json")
	}
	ctx.Exec([]string{"cp", vcpkgJSONSourceFilename, vcpkgJSONDestinationFilename})

	installLayer := ctx.Layer(installLayerName, gcp.LaunchLayer)

	vcpkgExePath := filepath.Join(vcpkgPath, "vcpkg")
	cmakeExePath, err := getToolPath(ctx, vcpkgExePath, "cmake")
	if err != nil {
		return err
	}
	ninjaExePath, err := getToolPath(ctx, vcpkgExePath, "ninja")
	if err != nil {
		return err
	}

	// vcpkg is not retrying downloads at this time. Do that manually.
	for i := 1; i < 32; i *= 2 {
		if err := warmupVcpkg(ctx, vcpkgExePath); err == nil {
			break
		}
		ctx.Logf("Downloading basic dependencies failed [%v], retrying in %d seconds...", err, i)
		time.Sleep(time.Duration(i) * time.Second)
	}

	args := []string{
		cmakeExePath,
		"-GNinja",
		"-DMAKE_BUILD_TYPE=Release",
		"-DCMAKE_CXX_COMPILER=g++-8",
		"-DCMAKE_C_COMPILER=gcc-8",
		fmt.Sprintf("-DCMAKE_MAKE_PROGRAM=%s", ninjaExePath),
		"-S", mainLayer.Path,
		"-B", buildLayer.Path,
		fmt.Sprintf("-DCNB_APP_DIR=%s", ctx.ApplicationRoot()),
		fmt.Sprintf("-DCMAKE_INSTALL_PREFIX=%s", installLayer.Path),
		fmt.Sprintf("-DVCPKG_TARGET_TRIPLET=%s", vcpkgTripletName),
		fmt.Sprintf("-DCMAKE_TOOLCHAIN_FILE=%s/scripts/buildsystems/vcpkg.cmake", vcpkgPath),
		fmt.Sprintf("-DVCPKG_DEFAULT_BINARY_CACHE=%s", vcpkgCache.Path),
	}
	ctx.Exec(args, gcp.WithUserAttribution)

	ctx.Exec([]string{cmakeExePath, "--build", buildLayer.Path, "--target", "install"}, gcp.WithUserAttribution)

	ctx.AddWebProcess([]string{filepath.Join(installLayer.Path, "bin", "function")})
	return nil
}

func warmupVcpkg(ctx *gcp.Context, vcpkgExePath string) error {
	exec, err := ctx.ExecWithErr([]string{vcpkgExePath, "install", "--only-downloads", "functions-framework-cpp"}, gcp.WithUserAttribution)
	if err != nil {
		return fmt.Errorf("downloading sources (exit code %d): %v", exec.ExitCode, exec.Combined)
	}
	return nil
}

func getToolPath(ctx *gcp.Context, vcpkgExePath string, tool string) (string, error) {
	exec, err := ctx.ExecWithErr([]string{vcpkgExePath, "fetch", "--feature-flags=-manifests", tool}, gcp.WithUserAttribution)
	if err != nil {
		return "", fmt.Errorf("fetching %s tool path (exit code %d): %v", tool, exec.ExitCode, exec.Combined)
	}
	// Strip any trailing newline before returning
	return strings.TrimSuffix(exec.Stdout, "\n"), nil
}

func installVcpkg(ctx *gcp.Context) (string, error) {
	vcpkg := ctx.Layer(vcpkgLayerName, gcp.BuildLayer, gcp.CacheLayer)
	customTripletPath := filepath.Join(vcpkg.Path, "triplets", vcpkgTripletName+".cmake")
	vcpkgExePath := filepath.Join(vcpkg.Path, "vcpkg")
	// If the cache layer already has the right version, just reuse it.
	if ctx.FileExists(vcpkgExePath) && ctx.FileExists(customTripletPath) {
		version, err := getVcpkgVersion(ctx, vcpkgExePath)
		if err != nil {
			ctx.Debugf("getting vcpkg version %v", err)
		} else if version >= vcpkgToolVersion {
			ctx.CacheHit(vcpkgLayerName)
			return vcpkg.Path, nil
		}
	}
	ctx.CacheMiss(vcpkgLayerName)
	ctx.Logf("Installing vcpkg %s", vcpkgVersion)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", vcpkgURL, vcpkg.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	ctx.Exec([]string{filepath.Join(vcpkg.Path, "bootstrap-vcpkg.sh")})
	ctx.Exec([]string{"cp", filepath.Join(ctx.BuildpackRoot(), "converter", "x64-linux-nodebug.cmake"), customTripletPath})

	return vcpkg.Path, nil
}

func getVcpkgVersion(ctx *gcp.Context, vcpkgExePath string) (string, error) {
	exec, err := ctx.ExecWithErr([]string{vcpkgExePath, "version", "--feature-flags=-manifests"}, gcp.WithUserAttribution)
	if err != nil {
		return "", fmt.Errorf("fetching vcpkg version path (exit code %d, output %q): %v", exec.ExitCode, exec.Combined, err)
	}
	for _, line := range strings.Split(exec.Stdout, "\n") {
		if strings.HasPrefix(line, vcpkgVersionPrefix) {
			return strings.TrimPrefix(line, vcpkgVersionPrefix), nil
		}
	}
	return "", fmt.Errorf("cannot find version line in vcpkg version output: %s", exec.Combined)
}

func createMainCppFile(ctx *gcp.Context, fn fnInfo, main string) error {
	f := ctx.CreateFile(main)
	defer f.Close()

	tmpl := mainTmpl
	if err := tmpl.Execute(f, fn); err != nil {
		return fmt.Errorf("executing template: %v", err)
	}
	return nil
}

func extractFnInfo(ctx *gcp.Context) fnInfo {
	fnTarget := os.Getenv(env.FunctionTarget)
	fnSignature := os.Getenv(env.FunctionSignatureType)

	info := fnInfo{
		Target:    fnTarget,
		Namespace: "",
		ShortName: fnTarget,
		Signature: httpSignature,
	}
	if fnSignature == "cloudevent" {
		info.Signature = cloudEventSignature
	}

	c := strings.Split(fnTarget, "::")
	if len(c) != 1 {
		info.ShortName = c[len(c)-1]
		info.Namespace = strings.Join(c[:len(c)-1], "::")
	}

	return info
}
