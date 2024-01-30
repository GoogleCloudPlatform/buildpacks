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
	"strconv"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/Masterminds/semver"
)

var (
	// nextJsVersionKey is the metadata key used to store the nextjs build adaptor version in the nextjs layer.
	nextJsVersionKey = "version"
)

// InstallNextJsBuildAdaptor installs the nextjs build adaptor in the given layer if it is not already cached.
func InstallNextJsBuildAdaptor(ctx *gcp.Context, njsl *libcnb.Layer, njsVersion string) error {
	layerName := njsl.Name
	version := detectNextjsAdaptorVersion(njsVersion)

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

// detectNextjsAdaptorVersion determines the version of Nextjs that is needed by a nextjs project
func detectNextjsAdaptorVersion(njsVersion string) string {
	if version, err := semver.StrictNewVersion(njsVersion); err == nil {
		// match major + minor versions with the Nextjs version if Nextjs version is concrete
		adapterVersion := strconv.FormatUint(version.Major(), 10) + "." + strconv.FormatUint(version.Minor(), 10)
		return adapterVersion
	}
	constraint, err := semver.NewConstraint(njsVersion)
	if err != nil {
		return "latest"
	}
	var newConstraints []string
	for _, constraint := range strings.Split(constraint.String(), " ") {
		versionSplit := strings.Split(constraint, ".")

		if len(versionSplit) == 3 {
			// converts < into <= when patch version is greater than 0
			// this is needed since the patch version is being dropped
			// i.e.  <14.0.15 -> needs to map to <=14.0 instead of <14.0 to be equivalent
			if strings.HasPrefix(versionSplit[0], "<") && !strings.HasPrefix(versionSplit[0], "<=") &&
				!strings.HasPrefix(versionSplit[len(versionSplit)-1], "0") {
				versionSplit[0] = "<=" + versionSplit[0][1:]
			}
			// > needs to be converted to >= for the same reason
			if strings.HasPrefix(versionSplit[0], ">") && !strings.HasPrefix(versionSplit[0], ">=") {
				versionSplit[0] = ">=" + versionSplit[0][1:]
			}
			// strip off patch version
			versionSplit = versionSplit[:len(versionSplit)-1]
		}
		newConstraints = append(newConstraints, strings.Join(versionSplit, "."))
	}
	return strings.Join(newConstraints, " ")
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
