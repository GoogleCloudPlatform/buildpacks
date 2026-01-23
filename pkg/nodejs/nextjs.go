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

const (
	// NextJsVersionKey is the metadata key used to store the nextjs build adaptor version in the nextjs layer.
	NextJsVersionKey = "version"
)

// InstallNextJsBuildAdapter installs the nextjs build adaptor in the given layer if it is not already cached.
func InstallNextJsBuildAdapter(ctx *gcp.Context, njsl *libcnb.Layer) error {
	version, err := nextjsAdapterVersion(ctx)
	if err != nil {
		return gcp.InternalErrorf("failed to resolve latest nextjs adapter version: %w", err)
	}

	// Check the metadata in the cache layer to determine if version is already installed.
	metaVersion := ctx.GetMetadata(njsl, NextJsVersionKey)
	if version == metaVersion {
		ctx.CacheHit(njsl.Name)
		ctx.Logf("nextjs adaptor cache hit: %q, %q, skipping installation.", version, metaVersion)
		return nil
	}

	ctx.CacheMiss(njsl.Name)
	if err := ctx.ClearLayer(njsl); err != nil {
		return fmt.Errorf("clearing layer %q: %w", njsl.Name, err)
	}
	ctx.Logf("Installing nextjs adaptor %s", version)
	// TODO(b/323280044) account for different versions
	if _, err := ctx.Exec([]string{"npm", "install", "--prefix", njsl.Path, "@apphosting/adapter-nextjs@" + version}); err != nil {
		return err
	}
	// Store layer flags and metadata.
	ctx.SetMetadata(njsl, NextJsVersionKey, version)
	return nil
}

// nextjsAdapterVersion determines the latest version of the Nextjs build adaptor.
func nextjsAdapterVersion(ctx *gcp.Context) (string, error) {
	// TODO(b/323280044) account for different MAJOR.MINOR versions once development is more stable.
	version, err := ctx.Exec([]string{"npm", "view", "@apphosting/adapter-nextjs", "version"})
	if err != nil {
		return "", gcp.InternalErrorf("npm view failed: %w", err)
	}
	if version.Stdout == "" {
		return "", gcp.InternalErrorf("npm view failed: no output")
	}
	return version.Stdout, nil
}

// OverrideNextjsBuildScript overrides the build script to be the Nextjs build script
func OverrideNextjsBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-nextjs-build", njsl.Path))
}
