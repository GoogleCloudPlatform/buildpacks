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
	"errors"
	"fmt"
	"io/ioutil"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	tpc "github.com/GoogleCloudPlatform/buildpacks/pkg/tpc"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

// InstallerCapability is the key used to inject the RuntimeInstaller capability.
const InstallerCapability = "runtime.Installer"

// Installer defines the interface for installing language runtimes.
// This interface allows abstracting the installation logic so that it can be swapped out
// for the "maker" use case. In the standard build process, the implementation downloads
// and installs the runtime. In the maker process, it simply resolves the version and records it.
type Installer interface {
	InstallTarballIfNotCached(ctx *gcp.Context, runtime InstallableRuntime, versionConstraint string, layer *libcnb.Layer) (bool, error)
}

var (
	dartSdkURL         = "https://storage.googleapis.com/dart-archive/channels/stable/release/%s/sdk/dartsdk-linux-x64-release.zip"
	flutterSdkURL      = "https://storage.googleapis.com/flutter_infra_release/releases/%s"
	googleTarballURL   = "https://dl.google.com/runtimes/%s/%[2]s/%[2]s-%s.tar.gz"
	runtimeVersionsURL = "https://dl.google.com/runtimes/%s/%s/version.json"
	// goTarballURL is the location from which we download Go. This is different from other runtimes
	// because the Go team already provides re-built tarballs on the same CDN.
	goTarballURL          = "https://dl.google.com/go/go%s.linux-amd64.tar.gz"
	runtimeImageARRepoURL = "%s/%s/runtimes-%s/%s"
	runtimeImageARURL     = runtimeImageARRepoURL + ":%s"
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
	Jetty        InstallableRuntime = "jetty"

	Ubuntu1804 string = "ubuntu1804"
	Ubuntu2204 string = "ubuntu2204"
	Ubuntu2404 string = "ubuntu2404"

	tarballRegistryProdGae        = "gae-runtimes"
	tarballRegistryProdServerless = "serverless-runtimes"
	tarballRegistryQual           = "serverless-runtimes-qa"
	tarballRegistryDev            = "lorry"
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
	Jetty:     "Jetty",
}

// stackToOS contains the mapping of Stack to OS.
var stackToOS = map[string]string{
	"google":                 Ubuntu1804,
	"google.gae.18":          Ubuntu1804,
	"google.18":              Ubuntu1804,
	"google.22":              Ubuntu2204,
	"google.gae.22":          Ubuntu2204,
	"google.min.22":          Ubuntu2204,
	"firebase.apphosting.22": Ubuntu2204,
	"google.24":              Ubuntu2404,
	"google.24.full":         Ubuntu2404,
}

var languageRuntimes = []InstallableRuntime{Nodejs, PHP, Python, Ruby, OpenJDK, CanonicalJDK, Go, DotnetSDK, AspNetCore}

const (
	versionKey = "version"
	stackKey   = "stack"
	// localRuntimeVersionLabel is the label key for the local runtime version.
	// This avoids conflict with the label 'runtime_version' added by the
	// runtime buildpack.
	localRuntimeVersionLabel = "local_runtime_version"
	// languageNameLabel is the label key for the language name.
	languageNameLabel = "language_name"
	// gcpUserAgent is required for the Ruby runtime, but used for others for simplicity.
	gcpUserAgent = "GCPBuildpacks"
)

var localRuntimeVersionCmds = map[InstallableRuntime][]string{
	Python: {"python3", "-c", "import platform; print(platform.python_version())"},
	Nodejs: {"node", "-p", "process.versions.node"},
}

var errUnsupportedRuntime = errors.New("unsupported runtime for local check")

// OSForStack returns the Operating System being used by input stackID.
func OSForStack(ctx *gcp.Context) string {
	os, ok := stackToOS[ctx.StackID()]
	if !ok {
		ctx.Warnf("unknown stack ID %q, falling back to Ubuntu 24.04", ctx.StackID())
		os = Ubuntu2404
	}
	return os
}

// arHostname returns the Artifact Registry hostname for a given region, handling both GDU and TPC.
func arHostname(ctx *gcp.Context, region string) (string, error) {
	if tpc.IsTPC() {
		hostname, present := tpc.GetHostname()
		if !present {
			return "", gcp.InternalErrorf("failed to get hostname for TPC build")
		}
		return hostname, nil
	}
	// GDU case
	return fmt.Sprintf("%s-docker.pkg.dev", region), nil
}

func tarballRegistry() string {
	buildEnv := os.Getenv(env.BuildEnv)

	if tpcProject, present := tpc.GetTarballProject(); present {
		return tpcProject
	}

	switch buildEnv {
	case "qual":
		return tarballRegistryQual
	case "dev":
		return tarballRegistryDev
	default: // prod and any other case
		flag, present := os.LookupEnv(env.ServerlessRuntimesTarballs)
		if present && flag == "true" {
			return tarballRegistryProdServerless
		}
		return tarballRegistryProdGae
	}
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

// InstallFlutterSDK downloads a given version of the Flutter SDK to the specified layer.
func InstallFlutterSDK(ctx *gcp.Context, layer *libcnb.Layer, version string, archive string) error {
	if err := ctx.ClearLayer(layer); err != nil {
		return fmt.Errorf("clearing layer %q: %w", layer.Name, err)
	}
	sdkURL := fmt.Sprintf(flutterSdkURL, archive)

	tar, err := ioutil.TempFile(layer.Path, "flutter_linux*.tar.xz")
	if err != nil {
		return err
	}
	defer os.Remove(tar.Name())

	if err := fetch.GetURL(sdkURL, tar); err != nil {
		ctx.Warnf("Failed to download Flutter SDK from %s. You can specify the verison by setting the GOOGLE_RUNTIME_VERSION environment variable", sdkURL)
		return err
	}

	if _, err := ctx.Exec([]string{"tar", "xJf", tar.Name(), "--strip-components=1", "-C", layer.Path}); err != nil {
		return fmt.Errorf("extracting Flutter SDK: %v", err)
	}

	if _, err := ctx.Exec([]string{"bin/flutter", "doctor"}, gcp.WithWorkDir(layer.Path)); err != nil {
		return fmt.Errorf("extracting Flutter SDK: %v", err)
	}

	if _, err := ctx.Exec([]string{"bin/flutter", "precache", "--web"}, gcp.WithWorkDir(layer.Path)); err != nil {
		return fmt.Errorf("extracting Flutter SDK: %v", err)
	}

	ctx.SetMetadata(layer, stackKey, ctx.StackID())
	ctx.SetMetadata(layer, versionKey, version)

	return nil
}

// InstallTarballIfNotCached installs a runtime tarball hosted on dl.google.com into the provided layer
// with caching.
// Returns true if a cached layer is used.
func InstallTarballIfNotCached(ctx *gcp.Context, runtime InstallableRuntime, versionConstraint string, layer *libcnb.Layer) (bool, error) {
	if cap := ctx.Capability(InstallerCapability); cap != nil {
		if ri, ok := cap.(Installer); ok {
			return ri.InstallTarballIfNotCached(ctx, runtime, versionConstraint, layer)
		}
	}
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

	stripComponents := 0
	if runtime == OpenJDK || runtime == Go || runtime == Jetty {
		stripComponents = 1
	}

	registry := tarballRegistry()
	region, present := os.LookupEnv(env.RuntimeImageRegion)

	// TODO(b/466126787): Add Golang support for TPC. Combine this block with the else condition once Golang support is added.
	if tpc.IsTPC() {
		hostname, err := arHostname(ctx, region)
		if err != nil {
			return false, err
		}
		url := runtimeImageURL(hostname, registry, osName, runtime, version)
		if err := fetch.ARImage(url, "", layer.Path, stripComponents, ctx); err != nil {
			ctx.Warnf("Failed to download %s version %s osName %s from artifact registry. You can specify the version by setting the GOOGLE_RUNTIME_VERSION environment variable", runtimeName, version, osName)
			return false, err
		}
	} else if registry == tarballRegistryDev || runtime == Go || !present {
		// Use Lorry for dev env, Go runtime, or if the region is not set.
		runtimeURL := tarballDownloadURL(runtime, osName, version)
		if err := fetch.Tarball(runtimeURL, layer.Path, stripComponents); err != nil {
			ctx.Warnf("Failed to download %s version %s osName %s from lorry. You can specify the version by setting the GOOGLE_RUNTIME_VERSION environment variable. To use Artifact Registry instead, set the GOOGLE_RUNTIME_IMAGE_REGION environment variable (e.g., 'us').", runtimeName, version, osName)
			return false, err
		}
	} else {
		// Use Artifact Registry for other cases.
		hostname, err := arHostname(ctx, region)
		if err != nil {
			return false, err
		}
		url := runtimeImageURL(hostname, registry, osName, runtime, version)

		fallbackHostname, err := arHostname(ctx, fallbackRegion)
		if err != nil {
			// Fallback region should always be valid in GDU.
			return false, gcp.InternalErrorf("failed to get hostname for fallback region %s: %w", fallbackRegion, err)
		}
		fallbackURL := runtimeImageURL(fallbackHostname, registry, osName, runtime, version)

		if err := fetch.ARImage(url, fallbackURL, layer.Path, stripComponents, ctx); err != nil {
			ctx.Warnf("Failed to download %s version %s osName %s from artifact registry. You can specify the version by setting the GOOGLE_RUNTIME_VERSION environment variable", runtimeName, version, osName)
			return false, err
		}
	}

	ctx.SetMetadata(layer, stackKey, ctx.StackID())
	ctx.SetMetadata(layer, versionKey, version)

	return false, nil
}

func runtimeImageURL(hostname, registry, osName string, runtime InstallableRuntime, version string) string {
	return fmt.Sprintf(runtimeImageARURL, hostname, registry, osName, runtime, version)
}

func tarballDownloadURL(runtime InstallableRuntime, os, version string) string {
	if runtime == Go {
		return fmt.Sprintf(goTarballURL, version)
	}
	return fmt.Sprintf(googleTarballURL, os, runtime, strings.ReplaceAll(version, "+", "_"))
}

// rubyGemsAndBundlerVersion returns the rubygems and bundler2 versions to use based on the Ruby version.
func rubyGemsAndBundlerVersion(version string) (string, string) {
	rubygemsVersion := "3.3.15"
	bundler2Version := "2.1.4"

	// Older 2.x Ruby versions have been using RubyGems 3.1.2 on GAE/GCF.
	if strings.HasPrefix(version, "2.") {
		rubygemsVersion = "3.1.2"
	}
	// Ruby 3.0 has been using 3.2.26 on GAE/GCF
	if strings.HasPrefix(version, "3.0") {
		rubygemsVersion = "3.2.26"
	}

	if strings.HasPrefix(version, "4.") {
		rubygemsVersion = "4.0.3"
		bundler2Version = "4.0.3"
	}

	return rubygemsVersion, bundler2Version
}

func installBundler1(version string) bool {
	return strings.HasPrefix(version, "2.") || strings.HasPrefix(version, "3.0")
}

// PinGemAndBundlerVersion pins the RubyGems versions for GAE and GCF runtime versions to prevent
// unexpected behaviors with new versions. This is only expected to be called if the target
// platform is GAE or GCF.
func PinGemAndBundlerVersion(ctx *gcp.Context, version string, layer *libcnb.Layer) error {
	bundler1Version := "1.17.3"
	rubygemsVersion, bundler2Version := rubyGemsAndBundlerVersion(version)

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
	if installBundler1(version) {
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

func normalizeJDKVersion(v string) string {
	return strings.NewReplacer("_", "+", "-beta", "").Replace(v)
}

func getNormalizedVersionsMap(versions []string, runtime InstallableRuntime) map[string]string {
	normalizedMap := make(map[string]string)
	isJdk := runtime == OpenJDK || runtime == CanonicalJDK
	for _, v := range versions {
		if isJdk {
			normalizedV := normalizeJDKVersion(v)
			normalizedMap[normalizedV] = v
		} else {
			normalizedMap[v] = v
		}
	}
	return normalizedMap
}

// resolveVersionFromList returns the newest available version from a list of versions that
// satisfies the provided version constraint.
func resolveVersionFromList(versions []string, runtime InstallableRuntime, normalizedConstraint string) (string, error) {
	normalizedMap := getNormalizedVersionsMap(versions, runtime)
	v, err := version.ResolveVersion(normalizedConstraint, slices.Collect(maps.Keys(normalizedMap)))
	if err != nil {
		return "", err
	}
	return normalizedMap[v], nil
}

// tryResolveVersionFromRegion attempts to resolve a version from a given Artifact Registry region.
// It returns the resolved version string on success.
// It returns an error if fetching versions fails or if no version matches the constraint.
func tryResolveVersionFromRegion(ctx *gcp.Context, runtime InstallableRuntime, normalizedConstraint, osName, region, registry string) (string, error) {
	hostname, err := arHostname(ctx, region)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf(runtimeImageARRepoURL, hostname, registry, osName, runtime)
	versions, err := fetch.ARVersions(url, "", ctx)
	if err != nil {
		return "", err
	}
	return resolveVersionFromList(versions, runtime, normalizedConstraint)
}

func getRegionsToTry(ctx *gcp.Context, region string) ([]string, error) {
	if tpc.IsTPC() {
		return []string{region}, nil
	}
	return []string{region, fallbackRegion}, nil
}

// ResolveVersion returns the newest available version of a runtime that satisfies the provided
// version constraint.
func ResolveVersion(ctx *gcp.Context, runtime InstallableRuntime, verConstraint, osName string) (string, error) {
	if runtime == Go {
		// verConstraint will be resolved to the exact version in golang.RuntimeVersion() already.
		return verConstraint, nil
	}
	if runtime == Jetty {
		return "latest", nil
	}
	// Some release candidates do not follow the convention for semver
	// Specifically php. example - 8.3.0RC4.
	if IsReleaseCandidate(verConstraint) || version.IsExactSemver(verConstraint) {
		ctx.Logf("Using exact version %s for %s", verConstraint, runtimeNames[runtime])
		return verConstraint, nil
	}

	registry := tarballRegistry()
	region, isARrequest := os.LookupEnv(env.RuntimeImageRegion)

	normalizedConstraint := verConstraint
	if runtime == OpenJDK || runtime == CanonicalJDK {
		normalizedConstraint = normalizeJDKVersion(verConstraint)
	}

	if registry == tarballRegistryDev || !isARrequest {
		// Use Lorry for dev env or if the region is not set.
		url := fmt.Sprintf(runtimeVersionsURL, osName, runtime)
		var versions []string
		if err := fetch.JSON(url, &versions); err != nil {
			return "", gcp.InternalErrorf("fetching %s versions for %s: %v", runtimeNames[runtime], osName, err)
		}

		v, err := resolveVersionFromList(versions, runtime, normalizedConstraint)
		if err != nil {
			return "", gcp.InternalErrorf("invalid %s version specified: %v. You may need to use a different builder. Please check if the language version specified is supported by the os: %v. You can refer to https://cloud.google.com/docs/buildpacks/builders for a list of compatible runtime languages per builder", runtimeNames[runtime], err, osName)
		}
		return v, nil

	}

	// Use Artifact Registry.
	regionsToTry, err := getRegionsToTry(ctx, region)
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, r := range regionsToTry {
		v, err := tryResolveVersionFromRegion(ctx, runtime, normalizedConstraint, osName, r, registry)
		if err == nil {
			ctx.Logf("Resolved version %s for %s from region %s", v, runtimeNames[runtime], r)
			return v, nil
		}
		lastErr = err
		ctx.Logf("Failed to resolve version %s for %s from region %s: %v", verConstraint, runtimeNames[runtime], r, err)
	}

	// Failed to resolve in all regions.
	return "", gcp.InternalErrorf("invalid %s version specified: %v . Version constraint %q not satisfied by any available versions in Artifact Registry. You may need to use a different builder. Please check if the language version specified is supported. You can refer to https://cloud.google.com/docs/buildpacks/builders for a list of compatible runtime languages per builder", runtimeNames[runtime], lastErr, verConstraint)
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

func checkLocalRuntimeVersion(ctx *gcp.Context, runtime InstallableRuntime) (string, error) {
	// TODO(b/481611857): Support other runtimes for local version check.
	cmd, ok := localRuntimeVersionCmds[runtime]
	if !ok {
		return "", fmt.Errorf("%s: %w", runtime, errUnsupportedRuntime)
	}

	result, err := ctx.Exec(cmd, gcp.WithUserAttribution)
	if err != nil {
		return "", fmt.Errorf("checking local %s version: %w", runtime, err)
	}
	version := strings.TrimSpace(result.Stdout)
	if version == "" {
		return "", fmt.Errorf("local %s version check returned empty string", runtime)
	}
	return version, nil
}

// MakerInstaller implements the Installer interface for the maker tool.
// Instead of downloading and installing the runtime (which is heavy and unnecessary for
// generating the maker output), it simply resolves the version and records it as a label.
// This allows the maker tool to capture the resolved runtime version quickly.
type MakerInstaller struct{}

// InstallTarballIfNotCached for the maker tool only resolves the runtime version (locally if supported)
// and adds a label. It bypasses the actual download and installation of the runtime tarball.
func (mi MakerInstaller) InstallTarballIfNotCached(ctx *gcp.Context, runtime InstallableRuntime, versionConstraint string, layer *libcnb.Layer) (bool, error) {
	version, err := checkLocalRuntimeVersion(ctx, runtime)
	if err != nil {
		if errors.Is(err, errUnsupportedRuntime) {
			return false, gcp.UserErrorf("runtime %s is not supported by the maker tool", runtime)
		}
		return false, gcp.UserErrorf("failed to detect local %s runtime version: %v. Please ensure it is installed and in your PATH", runtime, err)
	}

	// For the maker use case, we only need to resolve the version and add it as a label.
	// We do not need to download or install the runtime tarball.
	ctx.AddLabel(localRuntimeVersionLabel, version)
	ctx.AddLabel(languageNameLabel, string(runtime))
	return false, nil
}
