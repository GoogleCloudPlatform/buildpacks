// Package commonbuildpacks provides a function to get a map of common buildpacks.
package commonbuildpacks

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	configentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/config/entrypoint/lib"
	configflex "github.com/GoogleCloudPlatform/buildpacks/cmd/config/flex/lib"
	utilsarchivesource "github.com/GoogleCloudPlatform/buildpacks/cmd/utils/archive_source/lib"
	utilslabelimage "github.com/GoogleCloudPlatform/buildpacks/cmd/utils/label/lib"
)

// CommonBuildpacks returns a map of common buildpacks that are used by all language runtimes.
func CommonBuildpacks() map[string]gcp.BuildpackFuncs {
	return map[string]gcp.BuildpackFuncs{
		"google.config.entrypoint": {
			Detect: configentrypoint.DetectFn,
			Build:  configentrypoint.BuildFn,
		},
		"google.config.flex": {
			Detect: configflex.DetectFn,
			Build:  configflex.BuildFn,
		},
		"google.utils.archive-source": {
			Detect: utilsarchivesource.DetectFn,
			Build:  utilsarchivesource.BuildFn,
		},
		"google.utils.label-image": {
			Detect: utilslabelimage.DetectFn,
			Build:  utilslabelimage.BuildFn,
		},
	}
}
