// Copyright 2022 Google LLC
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

package runtime

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

var (
	dartSdkURL         = "https://storage.googleapis.com/dart-archive/channels/stable/release/%s/sdk/dartsdk-linux-x64-release.zip"
	googleTarballURL   = "https://dl.google.com/runtimes/%s/%[2]s/%[2]s-%s.tar.gz"
	runtimeVersionsURL = "https://dl.google.com/runtimes/%s/%s/version.json"
	// goTarballURL is the location from which we download Go. This is different from other runtimes
	// because the Go team already provides re-built tarballs on the same CDN.
	goTarballURL          = "https://dl.google.com/go/go%s.linux-amd64.tar.gz"
	runtimeImageARURL     = "%s-docker.pkg.dev/gae-runtimes/runtimes-%s/%s:%s"
	runtimeImageARRepoURL = "%s-docker.pkg.dev/gae-runtimes/runtimes-%s/%s"
	fallbackRegion        = "us"
)

// InstallableRuntime is used to hold runtimes information
type InstallableRuntime string

// All runtimes that can be installed using the InstallTarballIfNotCached function.
const (
	Nodejs       InstallableRuntime = "nodejs"
	PHP          InstallableRuntime = "php"
	Python       InstallableRuntime = "python"
	Ruby         InstallableRuntime = "ruby"
	Nginx        InstallableRuntime = "nginx"
	Pid1         InstallableRuntime = "pid1"
	DotnetSDK    InstallableRuntime = "dotnetsdk"
	AspNetCore   InstallableRuntime = "aspnetcore"
	OpenJDK      InstallableRuntime = "openjdk"
	CanonicalJDK InstallableRuntime = "canonicaljdk"
	Go           InstallableRuntime = "go"

	ubuntu1804 string = "ubuntu1804"
	ubuntu2204 string = "ubuntu2204"
)

// User friendly display name of all runtime (e.g. for use in error message).
var runtimeNames = map[InstallableRuntime]string{
	Nodejs:    "Node.js",
	PHP:       "PHP Runtime",
	Python:    "Python",
	Ruby:      "Ruby Runtime",
	Nginx:     "Nginx Web Server",
	Pid1:      "Pid1",
	DotnetSDK: ".NET SDK",
	Go:        "Go",
}

// stackToOS contains the mapping of Stack to OS.
var stackToOS = map[string]string{
	"google":                 ubuntu1804,
	"google.gae.18":          ubuntu1804,
	"google.22":              ubuntu2204,
	"google.gae.22":          ubuntu2204,
	"google.min.22":          ubuntu2204,
	"firebase.apphosting.22": ubuntu2204,
}

var languageRuntimes = []InstallableRuntime{Nodejs, PHP, Python, Ruby, OpenJDK, CanonicalJDK, Go, DotnetSDK, AspNetCore}

const (
	versionKey = "version"
	stackKey   = "stack"
	// gcpUserAgent is required for the Ruby runtime, but used for others for simplicity.
	gcpUserAgent = "GCPBuildpacks"
)

// OSForStack returns the Operating System being used by input stackID.
func OSForStack(ctx *gcp.Context) string {
	os, ok := stackToOS[ctx.StackID()]
	if !ok {
		ctx.Warnf("unknown stack ID %q, falling back to Ubuntu 18.04", ctx.StackID())
		os = ubuntu1804
	}
	return os
}

// IsCached returns true if the requested version of a runtime is installed in the given layer.
func IsCached(ctx *gcp.Context, layer *libcnb.Layer, version string) bool {
	metaVersion := ctx.GetMetadata(layer, versionKey)
	metaStack := ctx.GetMetadata(layer, stackKey)
	return metaVersion == version && metaStack == ctx.StackID()
}

// InstallDartSDK downloads a given version of the dart SDK to the specified layer.
func InstallDartSDK(ctx *gcp.Context, layer *libcnb.Layer, version string) error {
	if err := ctx.ClearLayer(layer); err != nil {
		return fmt.Errorf("clearing layer %q: %w", layer.Name, err)
	}
	sdkURL := fmt.Sprintf(dartSdkURL, version)

	zip, err := ioutil.TempFile(layer.Path, "dart-sdk-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(zip.Name())

	if err := fetch.GetURL(sdkURL, zip); err != nil {
		ctx.Warnf("Failed to download Dart SDK from %s. You can specify the verison by setting the GOOGLE_RUNTIME_VERSION environment variable", sdkURL)
		return err
	}

	if _, err := ctx.Exec([]string{"unzip", "-q", zip.Name(), "-d", layer.Path}); err != nil {
		return fmt.Errorf("extracting Dart SDK: %v", err)
	}

	// Once extracted the SDK contents are in a subdirectory called "dart-sdk". We move everything up
	// one level so "bin" and "lib" end up in the layer path.
	files, err := ioutil.ReadDir(path.Join(layer.Path, "dart-sdk"))
	if err != nil {
		return err
	}
	for _, file := range files {
		op := path.Join(layer.Path, "dart-sdk", file.Name())
		np := path.Join(layer.Path, file.Name())
		if err := os.Rename(op, np); err != nil {
			return err
		}
	}

	ctx.SetMetadata(layer, stackKey, ctx.StackID())
	ctx.SetMetadata(layer, versionKey, version)

	return nil
}

// InstallTarballIfNotCached installs a runtime tarball hosted on dl.google.com into the provided layer
// with caching.
// Returns true if a cached layer is used.
func InstallTarballIfNotCached(ctx *gcp.Context, runtime InstallableRuntime, versionConstraint string, layer *libcnb.Layer) (bool, error) {
	runtimeName := runtimeNames[runtime]
	runtimeID := string(runtime)
	osName := OSForStack(ctx)

	version, err := ResolveVersion(ctx, runtime, versionConstraint, osName)
	if err != nil {
		return false, err
	}

	if err = ValidateFlexMinVersion(ctx, runtime, version); err != nil {
		return false, err
	}

	if layer.Cache {
		if IsCached(ctx, layer, version) {
			ctx.CacheHit(runtimeID)
			ctx.Logf("%s v%s cache hit, skipping installation.", runtimeName, version)
			return true, nil
		}
		ctx.CacheMiss(runtimeID)
	}

	if err := ctx.ClearLayer(layer); err != nil {
		return false, gcp.InternalErrorf("clearing layer %q: %w", layer.Name, err)
	}
	ctx.Logf("Installing %s v%s.", runtimeName, version)

	runtimeURL := tarballDownloadURL(runtime, osName, version)
	stripComponents := 0
	if runtime == OpenJDK || runtime == Go {
		stripComponents = 1
	}
	region, present := os.LookupEnv(env.RuntimeImageRegion)
	if present && runtime != Go {
		url := runtimeImageURL(runtime, osName, version, region)
		fallbackURL := runtimeImageURL(runtime, osName, version, fallbackRegion)
		if err := fetch.ARImage(url, fallbackURL, layer.Path, stripComponents, ctx); err != nil {
			ctx.Warnf("Failed to download %s version %s osName %s from artifact registry. You can specify the version by setting the GOOGLE_RUNTIME_VERSION environment variable", runtimeName, version, osName)
			return false, err
		}
	} else {
		if err := fetch.Tarball(runtimeURL, layer.Path, stripComponents); err != nil {
			ctx.Warnf("Failed to download %s version %s osName %s from lorry. You can specify the version by setting the GOOGLE_RUNTIME_VERSION environment variable", runtimeName, version, osName)
			return false, err
		}
	}

	ctx.SetMetadata(layer, stackKey, ctx.StackID())
	ctx.SetMetadata(layer, versionKey, version)

	return false, nil
}

func runtimeImageURL(runtime InstallableRuntime, osName, version, region string) string {
	flag, present := os.LookupEnv(env.ServerlessRuntimesTarballs)
	if present && flag == "true" {
		newRuntimeImageARURL := "%s-docker.pkg.dev/serverless-runtimes/runtimes-%s/%s:%s"
		return fmt.Sprintf(newRuntimeImageARURL, region, osName, runtime, version)
	}
	return fmt.Sprintf(runtimeImageARURL, region, osName, runtime, version)
}

func tarballDownloadURL(runtime InstallableRuntime, os, version string) string {
	if runtime == Go {
		return fmt.Sprintf(goTarballURL, version)
	}
	return fmt.Sprintf(googleTarballURL, os, runtime, strings.ReplaceAll(version, "+", "_"))
}

// PinGemAndBundlerVersion pins the RubyGems versions for GAE and GCF runtime versions to prevent
// unexpected behaviors with new versions. This is only expected to be called if the target
// platform is GAE or GCF.
func PinGemAndBundlerVersion(ctx *gcp.Context, version string, layer *libcnb.Layer) error {
	rubygemsVersion := "3.3.15"
	bundler1Version := "1.17.3"
	bundler2Version := "2.1.4"
	installBundler1 := false

	// Bundler 1 is only installed for older versions of Ruby
	// Older 2.x Ruby versions have been using RubyGems 3.1.2 on GAE/GCF.
	if strings.HasPrefix(version, "2.") {
		rubygemsVersion = "3.1.2"
		installBundler1 = true
	}
	// Ruby 3.0 has been using 3.2.26 on GAE/GCF
	if strings.HasPrefix(version, "3.0") {
		rubygemsVersion = "3.2.26"
		installBundler1 = true
	}

	rubyBinPath := filepath.Join(layer.Path, "bin")
	gemPath := filepath.Join(rubyBinPath, "gem")

	// Update RubyGems to a fixed version
	ctx.Logf("Installing RubyGems %s", rubygemsVersion)
	_, err := ctx.Exec(
		[]string{gemPath, "update", "--no-document", "--system", rubygemsVersion}, gcp.WithUserAttribution)
	if err != nil {
		return fmt.Errorf("updating rubygems %s, err: %v", rubygemsVersion, err)
	}

	// Remove any existing bundler versions in the Ruby installation
	command := []string{"rm", "-f",
		filepath.Join(rubyBinPath, "bundle"), filepath.Join(rubyBinPath, "bundler")}
	_, err = ctx.Exec(command, gcp.WithUserAttribution)
	if err != nil {
		return fmt.Errorf("removing out-of-box bundler: %v", err)
	}

	command = []string{gemPath, "install", "--no-document", fmt.Sprintf("bundler:%s", bundler2Version)}
	if installBundler1 {
		// Install fixed versions of Bundler1 and Bundler2 for backwards compatibility
		command = append(command, fmt.Sprintf("bundler:%s", bundler1Version))
		ctx.Logf("Installing bundler %s and %s", bundler1Version, bundler2Version)
	} else {
		ctx.Logf("Installing bundler %s ", bundler2Version)
	}
	_, err = ctx.Exec(command, gcp.WithUserAttribution)
	if err != nil {
		return fmt.Errorf("installing bundler %s and %s: %v", bundler1Version, bundler2Version, err)
	}
	return nil
}

// IsReleaseCandidate returns true if given string is a RC candidate version.
func IsReleaseCandidate(verConstraint string) bool {
	return version.IsReleaseCandidate(verConstraint)
}

// ResolveVersion returns the newest available version of a runtime that satisfies the provided
// version constraint.
func ResolveVersion(ctx *gcp.Context, runtime InstallableRuntime, verConstraint, osName string) (string, error) {
	if runtime == Go {
		// Go provides its own version manifest so it has its own version resolution logic.
		return golang.ResolveGoVersion(verConstraint)
	}
	// Some release candidates do not follow the convention for semver
	// Specifically php. example - 8.3.0RC4.
	if IsReleaseCandidate(verConstraint) || version.IsExactSemver(verConstraint) {
		return verConstraint, nil
	}

	var versions []string
	var err error
	region, present := os.LookupEnv(env.RuntimeImageRegion)
	if present {
		url := fmt.Sprintf(runtimeImageARRepoURL, region, osName, runtime)
		fallbackURL := fmt.Sprintf(runtimeImageARRepoURL, fallbackRegion, osName, runtime)
		versions, err = fetch.ARVersions(url, fallbackURL, ctx)
	} else {
		url := fmt.Sprintf(runtimeVersionsURL, osName, runtime)
		err = fetch.JSON(url, &versions)
	}
	if err != nil {
		return "", gcp.InternalErrorf("fetching %s versions %s osName: %v", runtimeNames[runtime], osName, err)
	}

	if present && (runtime == OpenJDK || runtime == CanonicalJDK) {
		for i, v := range versions {
			// When resolving version openjdk versions should be decoded to align with semver requirement. (eg. 11.0.21_9 -> 11.0.21+9)
			versions[i] = strings.ReplaceAll(v, "_", "+")
		}
	}
	v, err := version.ResolveVersion(verConstraint, versions)
	if err != nil {
		return "", gcp.UserErrorf("invalid %s version specified: %v. You may need to use a different builder. Please check if the language version specified is supported by the os: %v. You can refer to https://cloud.google.com/docs/buildpacks/builders for a list of compatible runtime languages per builder", runtimeNames[runtime], err, osName)
	}
	// When downloading from AR the openjdk version should be encoded to align with tag format requirement. (eg. 11.0.21+9 -> 11.0.21_9)
	if present {
		if runtime == OpenJDK || runtime == CanonicalJDK {
			v = strings.ReplaceAll(v, "+", "_")
		}
	}
	return v, nil
}

// ValidateFlexMinVersion validates the minimum flex version for a given runtime.
func ValidateFlexMinVersion(ctx *gcp.Context, runtime InstallableRuntime, version string) error {
	if !env.IsFlex() || !slices.Contains(languageRuntimes, runtime) {
		return nil
	}

	if !runtimeMatchesInstallableRuntime(runtime) {
		return nil
	}

	minVersionEnv, present := os.LookupEnv(env.FlexMinVersion)
	if !present {
		return nil
	}

	minVersion, err := semver.NewVersion(minVersionEnv)
	if err != nil {
		// Ignore the error if env version is incorrect since it should be set by RCS.
		return nil
	}
	currentVersion, err := semver.NewVersion(version)
	if err != nil {
		return err
	}

	if currentVersion.LessThan(minVersion) {
		return gcp.UserErrorf("flex version %s is less than the minimum version %s allowed", version, minVersionEnv)
	}

	return nil
}

// runtimeMatchesInstallableRuntime returns true if the GOOGLE_RUNTIME should match the installable runtime.
// This is because PHP might install python and ruby might install nodejs.
func runtimeMatchesInstallableRuntime(installableRuntime InstallableRuntime) bool {
	switch r := os.Getenv(env.Runtime); r {
	case "java":
		return installableRuntime == OpenJDK || installableRuntime == CanonicalJDK
	case "dotnet":
		return installableRuntime == DotnetSDK || installableRuntime == AspNetCore
	default:
		return strings.HasPrefix(r, string(installableRuntime))
	}
}
