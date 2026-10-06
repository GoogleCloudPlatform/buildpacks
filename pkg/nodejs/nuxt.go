package nodejs

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// NuxtStartCommand determines if this is a Nuxt application and returns the command to start the
// nuxt server. If not it is not a Nuxt application it returns nil.
func NuxtStartCommand(ctx *gcp.Context) ([]string, error) {
	configExists, err := ctx.FileExists(ctx.ApplicationRoot(), "nuxt.config.ts")
	if err != nil {
		return nil, err
	}
	serverExists, err := ctx.FileExists(ctx.ApplicationRoot(), ".output/server/index.mjs")
	if err != nil {
		return nil, err
	}
	if configExists && serverExists {
		return []string{"node", ".output/server/index.mjs"}, nil
	}
	return nil, nil
}
