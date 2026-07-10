package gcpbuildpack

import "maps"

// Register registers buildpack functions.
func Register(buildpacks map[string]BuildpackFuncs, registrations map[string]BuildpackFuncs) {
	maps.Copy(buildpacks, registrations)
}
