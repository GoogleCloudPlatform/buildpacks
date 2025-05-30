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
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

const (
	// AngularVersionKey is the metadata key used to store the angular build adapter version in the angular layer.
	AngularVersionKey = "version"
)

// InstallAngularBuildAdapter installs the angular build adaptor in the given layer if it is not already cached.
func InstallAngularBuildAdapter(ctx *gcp.Context, al *libcnb.Layer) error {
	version, err := angularAdapterVersion(ctx)
	if err != nil {
		return gcp.InternalErrorf("failed to resolve latest angular adapter version: %w", err)
	}

	// Check the metadata in the cache layer to determine if version is already installed.
	metaVersion := ctx.GetMetadata(al, AngularVersionKey)
	if version == metaVersion {
		ctx.CacheHit(al.Name)
		ctx.Logf("angular adaptor cache hit: %q, %q, skipping installation.", version, metaVersion)
		return nil
	}

	ctx.CacheMiss(al.Name)
	if err := ctx.ClearLayer(al); err != nil {
		return fmt.Errorf("clearing layer %q: %w", al.Name, err)
	}
	ctx.Logf("Installing angular adaptor %s", version)
	// TODO(b/323280044) account for different versions
	if _, err := ctx.Exec([]string{"npm", "install", "--prefix", al.Path, "@apphosting/adapter-angular@" + version}); err != nil {
		return err
	}
	// Store layer flags and metadata.
	ctx.SetMetadata(al, AngularVersionKey, version)
	return nil
}

// angularAdapterVersion determines the latest version of the Angular adapter.
func angularAdapterVersion(ctx *gcp.Context) (string, error) {
	// TODO(b/323280044) account for different MAJOR.MINOR versions once development is more stable.
	version, err := ctx.Exec([]string{"npm", "view", "@apphosting/adapter-angular", "version"})
	if err != nil {
		return "", gcp.InternalErrorf("npm view failed: %w", err)
	}
	if version.Stdout == "" {
		return "", gcp.InternalErrorf("npm view returned empty stdout")
	}
	return version.Stdout, nil
}

// OverrideAngularBuildScript overrides the build script to be the Angular build script.
func OverrideAngularBuildScript(njsl *libcnb.Layer) {
	njsl.BuildEnvironment.Override(AppHostingBuildEnv, fmt.Sprintf("npm exec --prefix %s apphosting-adapter-angular-build", njsl.Path))
}

// ExtractAngularStartCommand inspects the given package.json file for an idiomatic `serve:ssr:APP_NAME`
// command. If one exists, its value is returned. If not, return an empty string.
func ExtractAngularStartCommand(pjs *PackageJSON) string {
	for k, v := range pjs.Scripts {
		if strings.HasPrefix(k, "serve:ssr:") {
			return v
		}
	}
	return ""
}
