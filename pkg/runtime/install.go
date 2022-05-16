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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/buildpacks/libcnb"
)

var (
	dartSdkURL         = "https://storage.googleapis.com/dart-archive/channels/stable/release/%s/sdk/dartsdk-linux-x64-release.zip"
	googleTarballURL   = "https://dl.google.com/runtimes/%[1]s/%[1]s-%s.tar.gz"
	runtimeVersionsURL = "https://dl.google.com/runtimes/%s/version.json"
)

// InstallableRuntime is used to hold runtimes information
type InstallableRuntime string

// All runtimes that can be installed using the InstallTarballIfNotCached function.
const (
	Nodejs InstallableRuntime = "nodejs"
	PHP    InstallableRuntime = "php"
	Python InstallableRuntime = "python"
	Ruby   InstallableRuntime = "ruby"
	Nginx  InstallableRuntime = "nginx"
	Pid1   InstallableRuntime = "pid1"
	Serve  InstallableRuntime = "serve"
)

// User friendly display name of all runtime (e.g. for use in error message).
var runtimeNames = map[InstallableRuntime]string{
	Nodejs: "Node.js",
	PHP:    "PHP Runtime",
	Python: "Python",
	Ruby:   "Ruby Runtime",
	Nginx:  "Nginx Web Server",
	Pid1:   "Pid1",
	Serve:  "Serve",
}

const (
	versionKey = "version"
	// gcpUserAgent is required for the Ruby runtime, but used for others for simplicity.
	gcpUserAgent = "GCPBuildpacks"
)

// IsCached returns true if the requested version of a runtime is installed in the given layer.
func IsCached(ctx *gcp.Context, layer *libcnb.Layer, version string) bool {
	metaVersion := ctx.GetMetadata(layer, versionKey)
	return metaVersion == version
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

	if _, err := ctx.ExecWithErr([]string{"unzip", "-q", zip.Name(), "-d", layer.Path}); err != nil {
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

	ctx.SetMetadata(layer, versionKey, version)

	return nil
}

// InstallTarballIfNotCached installs a runtime tarball hosted on dl.google.com into the provided layer
// with caching.
// Returns true if a cached layer is used.
func InstallTarballIfNotCached(ctx *gcp.Context, runtime InstallableRuntime, versionConstraint string, layer *libcnb.Layer) (bool, error) {
	runtimeName := runtimeNames[runtime]
	runtimeID := string(runtime)

	version, err := resolveVersion(runtime, versionConstraint)
	if err != nil {
		return false, err
	}
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     runtimeID,
		Metadata: map[string]interface{}{"version": version},
		Launch:   true,
		Build:    true,
	})

	if IsCached(ctx, layer, version) {
		ctx.CacheHit(runtimeID)
		ctx.Logf("%s v%s cache hit, skipping installation.", runtimeName, version)
		return true, nil
	}
	ctx.CacheMiss(runtimeID)
	if err := ctx.ClearLayer(layer); err != nil {
		return false, gcp.InternalErrorf("clearing layer %q: %w", layer.Name, err)
	}
	ctx.Logf("Installing %s v%s.", runtimeName, version)

	ctx.SetMetadata(layer, versionKey, version)
	runtimeURL := fmt.Sprintf(googleTarballURL, runtime, version)

	if err := fetch.Tarball(runtimeURL, layer.Path, 0); err != nil {
		ctx.Warnf("Failed to download %s version %s. You can specify the verison by setting the GOOGLE_RUNTIME_VERSION environment variable", runtimeName, version)
		return false, err
	}

	ctx.SetMetadata(layer, versionKey, version)

	return false, nil
}

// resolveVersion returns the newest available version of a runtime that satisfies the provided
// version constraint.
func resolveVersion(runtime InstallableRuntime, verConstraint string) (string, error) {
	if version.IsExactSemver(verConstraint) {
		return verConstraint, nil
	}

	url := fmt.Sprintf(runtimeVersionsURL, runtime)

	var versions []string
	if err := fetch.JSON(url, &versions); err != nil {
		return "", gcp.InternalErrorf("fetching %s versions: %v", runtimeNames[runtime], err)
	}

	v, err := version.ResolveVersion(verConstraint, versions)
	if err != nil {
		return "", gcp.UserErrorf("invalid %s version specified: %v", runtimeNames[runtime], err)
	}
	return v, nil
}
