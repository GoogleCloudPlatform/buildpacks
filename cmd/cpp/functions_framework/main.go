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
	vcpkgTarballPrefix          = "https://github.com/microsoft/vcpkg/archive"
	vcpkgVersion                = "83bc451982951fb2a7fc19d75ae1ac9504816471"
	vcpkgVersionPrefix          = "Vcpkg package management program version "
	vcpkgTripletName            = "x64-linux-nodebug"
	installLayerName            = "cpp"
	functionsFrameworkNamespace = "::google::cloud::functions"
)

type signatureInfo struct {
	ReturnType   string
	ArgumentType string
	WrapperType  string
	Eval         string
}

var (
	vcpkgURL             = fmt.Sprintf("%s/%s.tar.gz", vcpkgTarballPrefix, vcpkgVersion)
	mainTmpl             = template.Must(template.New("mainV0").Parse(mainTextTemplateV0))
	declarativeSignature = signatureInfo{
		ReturnType:   functionsFrameworkNamespace + "::Function",
		ArgumentType: "",
		WrapperType:  "",
		Eval:         "()",
	}
	httpSignature = signatureInfo{
		ReturnType:   functionsFrameworkNamespace + "::HttpResponse",
		ArgumentType: functionsFrameworkNamespace + "::HttpRequest",
		WrapperType:  functionsFrameworkNamespace + "::UserHttpFunction",
		Eval:         "",
	}
	cloudEventSignature = signatureInfo{
		ReturnType:   "void",
		ArgumentType: functionsFrameworkNamespace + "::CloudEvent",
		WrapperType:  functionsFrameworkNamespace + "::UserCloudEventFunction",
		Eval:         "",
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

func hasCppCode(ctx *gcp.Context) (bool, error) {
	exists, err := ctx.FileExists("CMakeLists.txt")
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	for _, pattern := range []string{"*.cc", "*.cxx", "*.cpp"} {
		atLeastOne, err := ctx.HasAtLeastOne(pattern)
		if err != nil {
			return false, fmt.Errorf("finding %v files: %w", pattern, err)
		}
		if atLeastOne {
			return true, nil
		}
	}
	return false, nil
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	hasCpp, err := hasCppCode(ctx)
	if err != nil {
		return nil, err
	}
	if !hasCpp {
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

	vcpkgCache, err := ctx.Layer(vcpkgCacheLayerName, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", vcpkgCacheLayerName, err)
	}

	mainLayer, err := ctx.Layer(mainLayerName)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", mainLayerName, err)
	}
	if err := ctx.SetFunctionsEnvVars(mainLayer); err != nil {
		return err
	}

	buildLayer, err := ctx.Layer(buildLayerName, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", buildLayerName, err)
	}

	fn := extractFnInfo(os.Getenv(env.FunctionTarget), os.Getenv(env.FunctionSignatureType))
	if err := createMainCppFile(ctx, fn, filepath.Join(mainLayer.Path, "main.cc")); err != nil {
		return err
	}
	if err := createMainCppSupportFiles(ctx, mainLayer.Path, ctx.BuildpackRoot()); err != nil {
		return err
	}

	installLayer, err := ctx.Layer(installLayerName, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", installLayerName, err)
	}

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
	}
	ctx.Exec(args, gcp.WithUserAttribution, gcp.WithEnv(
		fmt.Sprintf("VCPKG_DEFAULT_BINARY_CACHE=%s", vcpkgCache.Path),
		fmt.Sprintf("VCPKG_DEFAULT_HOST_TRIPLET=%s", vcpkgTripletName)))
	ctx.Exec([]string{cmakeExePath, "--build", buildLayer.Path, "--target", "install"}, gcp.WithUserAttribution)

	ctx.AddWebProcess([]string{filepath.Join(installLayer.Path, "bin", "function")})
	return nil
}

func warmupVcpkg(ctx *gcp.Context, vcpkgExePath string) error {
	exec, err := ctx.ExecWithErr([]string{vcpkgExePath, "install", "--feature-flags=-manifests", "--only-downloads", "functions-framework-cpp"}, gcp.WithUserAttribution)
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
	// If the tool needs to be downloaded, vcpkg now prints additional informational messages before the actual path.
	// Ignore all these messages.
	ss := strings.Split(exec.Stdout, "\n")
	if len(ss) < 1 {
		return "", fmt.Errorf("fetching %s tool path, output should have at least one newline", tool)
	}
	return ss[len(ss)-1], nil
}

func installVcpkg(ctx *gcp.Context) (string, error) {
	vcpkg, err := ctx.Layer(vcpkgLayerName, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return "", fmt.Errorf("creating %v layer: %w", vcpkgLayerName, err)
	}
	customTripletPath := filepath.Join(vcpkg.Path, "triplets", vcpkgTripletName+".cmake")
	vcpkgExePath := filepath.Join(vcpkg.Path, "vcpkg")
	vcpkgBaselinePath := filepath.Join(vcpkg.Path, "versions", "baseline.json")
	isValid, err := validateVcpkgCache(ctx, customTripletPath, vcpkgExePath, vcpkgBaselinePath)
	if err != nil {
		return "", err
	}
	if isValid {
		ctx.CacheHit(vcpkgLayerName)
		return vcpkg.Path, nil
	}
	ctx.CacheMiss(vcpkgLayerName)
	ctx.Logf("Installing vcpkg %s", vcpkgVersion)
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", vcpkgURL, vcpkg.Path)
	ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution)

	ctx.Exec([]string{filepath.Join(vcpkg.Path, "bootstrap-vcpkg.sh")})
	ctx.Exec([]string{"cp", filepath.Join(ctx.BuildpackRoot(), "converter", "x64-linux-nodebug.cmake"), customTripletPath})

	return vcpkg.Path, nil
}

func validateVcpkgCache(ctx *gcp.Context, customTripletPath string, vcpkgExePath string, vcpkgBaselinePath string) (bool, error) {
	exists, err := ctx.FileExists(customTripletPath)
	if err != nil {
		return false, err
	}
	if !exists {
		ctx.Debugf("Missing vcpkg custom triplet (%s)", customTripletPath)
		return false, nil
	}
	exists, err = ctx.FileExists(vcpkgBaselinePath)
	if err != nil {
		return false, err
	}
	if !exists {
		ctx.Debugf("Missing vcpkg baseline file (%s)", vcpkgBaselinePath)
		return false, nil
	}
	exists, err = ctx.FileExists(vcpkgExePath)
	if err != nil {
		return false, err
	}
	if !exists {
		ctx.Debugf("Missing vcpkg tool (%s)", vcpkgExePath)
		return false, nil
	}
	return true, nil
}

func createMainCppFile(ctx *gcp.Context, fn fnInfo, main string) error {
	f, err := ctx.CreateFile(main)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl := mainTmpl
	if err := tmpl.Execute(f, fn); err != nil {
		return fmt.Errorf("executing template: %v", err)
	}
	return nil
}

func extractFnInfo(fnTarget string, fnSignature string) fnInfo {
	info := fnInfo{
		Target:    fnTarget,
		Namespace: "",
		ShortName: fnTarget,
		Signature: declarativeSignature,
	}
	if fnSignature == "http" {
		info.Signature = httpSignature
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

func createMainCppSupportFiles(ctx *gcp.Context, main string, buildpackRoot string) error {
	ctx.Exec([]string{"cp", filepath.Join(buildpackRoot, "converter", "CMakeLists.txt"), filepath.Join(main, "CMakeLists.txt")})

	vcpkgJSONDestinationFilename := filepath.Join(main, "vcpkg.json")
	vcpkgJSONSourceFilename := filepath.Join(ctx.ApplicationRoot(), "vcpkg.json")

	vcpkgExists, err := ctx.FileExists(vcpkgJSONSourceFilename)
	if err != nil {
		return err
	}
	if !vcpkgExists {
		vcpkgJSONSourceFilename = filepath.Join(buildpackRoot, "converter", "vcpkg.json")
	}
	ctx.Exec([]string{"cp", vcpkgJSONSourceFilename, vcpkgJSONDestinationFilename})

	return nil
}
