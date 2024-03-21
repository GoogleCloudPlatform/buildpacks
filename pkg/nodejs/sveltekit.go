package nodejs

import (
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// SvelteAdapterEnv is an env var that enables SvelteKit to detect
// Google Cloud Buildpacks and use adapter-node when adapter-auto is detected
// https://github.com/sveltejs/kit/blob/main/packages/adapter-auto/adapters.js
const SvelteAdapterEnv = "GCP_BUILDPACKS=TRUE"

// DetectSvelteKitAutoAdapter returns true if the given package.json file
// contains the @sveltejs/adapter-auto dependency and no others.
func DetectSvelteKitAutoAdapter(p *PackageJSON) bool {
	// Check and reject if the package contains an adapter that is
	// not the @sveltejs/adapter-auto dependency.
	for k := range p.DevDependencies {
		if strings.HasPrefix(k, "@sveltejs/adapter-") && k != "@sveltejs/adapter-auto" {
			return false
		}
	}
	_, ok := p.DevDependencies["@sveltejs/adapter-auto"]
	return ok
}

// SvelteKitStartCommand determines if this is a SvelteKit application and returns the command to start the
// SvelteKit server. If not it is not a SvelteKit application it returns nil.
func SvelteKitStartCommand(ctx *gcp.Context) ([]string, error) {
	configExists, err := ctx.FileExists(ctx.ApplicationRoot(), "svelte.config.js")
	if err != nil {
		return nil, err
	}
	serverExists, err := ctx.FileExists(ctx.ApplicationRoot(), "build/index.js")
	if err != nil {
		return nil, err
	}
	if configExists && serverExists {
		return []string{"node", "build/index.js"}, nil
	}
	return nil, nil
}
