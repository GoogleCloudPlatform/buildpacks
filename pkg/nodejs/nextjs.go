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
	// nextJsVersionKey is the metadata key used to store the nextjs build adaptor version in the nextjs layer.
	nextJsVersionKey = "version"
)

// InstallNextJsBuildAdaptor installs the nextjs build adaptor in the given layer if it is not already cached.
func InstallNextJsBuildAdaptor(ctx *gcp.Context, njsl *libcnb.Layer) error {
	layerName := njsl.Name
	version, err := detectNextjsAdaptorVersion()
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
		if err := downloadNextJsAdaptor(ctx, njsl.Path); err != nil {
			return gcp.InternalErrorf("downloading nextjs adapter: %w", err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(njsl, nextJsVersionKey, version)
	return nil
}

// detectNextjsAdaptorVersion determines the version of Nextjs that is needed by a nextjs project
func detectNextjsAdaptorVersion() (string, error) {
	// TODO(b/313959098) for now this will always be latest, later we will need to support different versions
	return "latest", nil
}

// downloadNextJsAdaptor downloads the nextjs build adaptor into the provided directory.
func downloadNextJsAdaptor(ctx *gcp.Context, dirPath string) error {
	// TODO(b/313959098) Account for different versions
	_, err := ctx.Exec([]string{"npm", "install", "--prefix", dirPath, "@apphosting/adapter-nextjs@latest"})
	if err != nil {
		return gcp.InternalErrorf("installing nextjs adaptor: %w", err)
	}
	return nil
}

// OverrideNextjsBuildScript overrides the build script to be the nextjs buildscript
func OverrideNextjsBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-nextjs-build", njsl.Path))
}
