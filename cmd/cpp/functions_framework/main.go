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
	"crypto/sha256"
	"fmt"
	"io"
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
	vcpkgVersion                = "dfcd4e4b30799c4ce02fe3939b62576fec444224"
	vcpkgBaselineSha256         = "7738af3dce5670a319f4812b95d05947ec1afcd0acdc3aa63df2078e0af2794f"
	vcpkgToolVersion            = "2021-08-12-unknownhash"
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

	fn := extractFnInfo(os.Getenv(env.FunctionTarget), os.Getenv(env.FunctionSignatureType))
	if err := createMainCppFile(ctx, fn, filepath.Join(mainLayer.Path, "main.cc")); err != nil {
		return err
	}
	if err := createMainCppSupportFiles(ctx, mainLayer.Path, ctx.BuildpackRoot()); err != nil {
		return err
	}

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
	}
	ctx.Exec(args, gcp.WithUserAttribution, gcp.WithEnv(fmt.Sprintf("VCPKG_DEFAULT_BINARY_CACHE=%s", vcpkgCache.Path)))

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
	// Strip any trailing newline before returning
	return strings.TrimSuffix(exec.Stdout, "\n"), nil
}

func installVcpkg(ctx *gcp.Context) (string, error) {
	vcpkg := ctx.Layer(vcpkgLayerName, gcp.BuildLayer, gcp.CacheLayer)
	customTripletPath := filepath.Join(vcpkg.Path, "triplets", vcpkgTripletName+".cmake")
	vcpkgExePath := filepath.Join(vcpkg.Path, "vcpkg")
	vcpkgBaselinePath := filepath.Join(vcpkg.Path, "versions", "baseline.json")
	if validateVcpkgCache(ctx, customTripletPath, vcpkgExePath, vcpkgBaselinePath) {
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

func validateVcpkgCache(ctx *gcp.Context, customTripletPath string, vcpkgExePath string, vcpkgBaselinePath string) bool {
	if !ctx.FileExists(customTripletPath) {
		ctx.Debugf("Missing vcpkg custom triplet (%s)", customTripletPath)
		return false
	}
	if !ctx.FileExists(vcpkgBaselinePath) {
		ctx.Debugf("Missing vcpkg baseline file (%s)", vcpkgBaselinePath)
		return false
	}
	if !ctx.FileExists(vcpkgExePath) {
		ctx.Debugf("Missing vcpkg tool (%s)", vcpkgExePath)
		return false
	}
	actualVcpkgToolVersion, err := getVcpkgToolVersion(ctx, vcpkgExePath)
	if err != nil {
		ctx.Debugf("Getting vcpkg version %v", err)
		return false
	}
	if actualVcpkgToolVersion != vcpkgToolVersion {
		ctx.Debugf("Mismatched vcpkg tool version, got=%s, want=%s", actualVcpkgToolVersion, actualVcpkgToolVersion)
		return false
	}
	actualVcpkgBaselineSha256, err := getVcpkgBaselineSha256(ctx, vcpkgBaselinePath)
	if err != nil {
		ctx.Debugf("Getting vcpkg baseline hash %v", err)
		return false
	}
	if actualVcpkgBaselineSha256 != vcpkgBaselineSha256 {
		ctx.Debugf("Mismatched vcpkg baseline SHA256, got=%s, want=%s", actualVcpkgBaselineSha256, vcpkgBaselineSha256)
		return false
	}
	return true
}

func getVcpkgToolVersion(ctx *gcp.Context, vcpkgExePath string) (string, error) {
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

func getVcpkgBaselineSha256(ctx *gcp.Context, vcpkgBaselinePath string) (string, error) {
	f, err := os.Open(vcpkgBaselinePath)
	if err != nil {
		return "", err
	}
	sha := sha256.New()
	if _, err := io.Copy(sha, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha.Sum(nil)), nil
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

func extractFnInfo(fnTarget string, fnSignature string) fnInfo {
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

func createMainCppSupportFiles(ctx *gcp.Context, main string, buildpackRoot string) error {
	ctx.Exec([]string{"cp", filepath.Join(buildpackRoot, "converter", "CMakeLists.txt"), filepath.Join(main, "CMakeLists.txt")})

	vcpkgJSONDestinationFilename := filepath.Join(main, "vcpkg.json")
	vcpkgJSONSourceFilename := filepath.Join(ctx.ApplicationRoot(), "vcpkg.json")

	if !ctx.FileExists(vcpkgJSONSourceFilename) {
		vcpkgJSONSourceFilename = filepath.Join(buildpackRoot, "converter", "vcpkg.json")
	}
	ctx.Exec([]string{"cp", vcpkgJSONSourceFilename, vcpkgJSONDestinationFilename})

	return nil
}
