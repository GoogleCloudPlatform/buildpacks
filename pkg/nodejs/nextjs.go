// Copyright 2023 Google LLC
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

package nodejs

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

var (
	// nextJsVersionKey is the metadata key used to store the nextjs build adaptor version in the nextjs layer.
	nextJsVersionKey = "version"
	// PinnedNextjsAdapterVersion is the version of the nextjs adapter that will be used.
	PinnedNextjsAdapterVersion = "14.0.9"
)

// InstallNextJsBuildAdaptor installs the nextjs build adaptor in the given layer if it is not already cached.
func InstallNextJsBuildAdaptor(ctx *gcp.Context, njsl *libcnb.Layer, njsVersion string) error {
	layerName := njsl.Name
	version, err := detectNextjsAdaptorVersion(njsVersion)

	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(njsl, nextJsVersionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("nextjs adaptor cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(njsl); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install nextjs adaptor in layer.
		ctx.Logf("Installing nextjs adaptor %s", version)
		if err := downloadNextJsAdaptor(ctx, njsl.Path, version); err != nil {
			return gcp.InternalErrorf("downloading nextjs adapter: %w", err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(njsl, nextJsVersionKey, version)
	return nil
}

// detectNextjsAdaptorVersion determines the version of Nextjs that is needed by a nextjs project.
func detectNextjsAdaptorVersion(njsVersion string) (string, error) {
	// TODO(b/323280044) account for different versions once development is more stable.
	adapterVersion := PinnedNextjsAdapterVersion
	return adapterVersion, nil
}

// downloadNextJsAdaptor downloads the Nextjs build adaptor into the provided directory.
func downloadNextJsAdaptor(ctx *gcp.Context, dirPath string, version string) error {
	if _, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-nextjs@" + version}); err != nil {
		ctx.Logf("Failed to install nextjs adaptor version: %s. Falling back to latest", version)
		if _, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-nextjs@latest"}); err != nil {
			return gcp.InternalErrorf("installing nextjs adaptor: %w", err)
		}
	}
	return nil
}

// OverrideNextjsBuildScript overrides the build script to be the Nextjs build script
func OverrideNextjsBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-nextjs-build", njsl.Path))
}
