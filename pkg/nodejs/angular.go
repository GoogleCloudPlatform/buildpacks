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
	"github.com/buildpacks/libcnb"
)

var (
	// angularVersionKey is the metadata key used to store the angular build adaptor version in the angular layer.
	angularVersionKey = "version"
)

// InstallAngularBuildAdaptor installs the angular build adaptor in the given layer if it is not already cached.
func InstallAngularBuildAdaptor(ctx *gcp.Context, njsl *libcnb.Layer) error {
	layerName := njsl.Name
	version, err := detectAngularAdaptorVersion()
	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(njsl, angularVersionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("angular adaptor cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(njsl); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install angular adaptor in layer.
		ctx.Logf("Installing angular adaptor %s", version)
		if err := downloadAngularAdaptor(ctx, njsl.Path); err != nil {
			return gcp.InternalErrorf("downloading angular adapter: %w", err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(njsl, angularVersionKey, version)
	return nil
}

// detectAngularAdaptorVersion determines the version of Angular that is needed by an Angular project
func detectAngularAdaptorVersion() (string, error) {
	// TODO(b/323280044) for now this will always be latest, later we will need to support different versions
	return "latest", nil
}

// downloadAngularAdaptor downloads the Angular build adaptor into the provided directory.
func downloadAngularAdaptor(ctx *gcp.Context, dirPath string) error {
	// TODO(b/323280044) account for different versions
	_, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-angular@latest"})
	if err != nil {
		return gcp.InternalErrorf("installing angular adaptor: %w", err)
	}
	return nil
}

// OverrideAngularBuildScript overrides the build script to be the Angular build script
func OverrideAngularBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-angular-build", njsl.Path))
}
