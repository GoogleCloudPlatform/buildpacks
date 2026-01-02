package nodejs

import (
	"fmt"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

var (
	// BunLock is the name of the Bun lock file.
	BunLock = "bun.lockb"
	// bunVersionKey is the metadata key used to store the Bun version in the bun layer.
	bunVersionKey = "version"
)

// InstallBun installs Bun in the given layer using the curl installer.
func InstallBun(ctx *gcp.Context, bunLayer *libcnb.Layer, pjs *PackageJSON) error {
	layerName := bunLayer.Name
	installDir := filepath.Join(bunLayer.Path, "bin")
	version, err := detectBunVersion(pjs)
	if err != nil {
		return err
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(bunLayer, bunVersionKey)
	if metaVersion == version {
		ctx.CacheHit(layerName)
		ctx.Logf("Bun v%s cache hit, skipping installation.", version)
	} else {
		ctx.CacheMiss(layerName)
		ctx.Logf("Installing Bun v%s.", version)
		if err := ctx.ClearLayer(bunLayer); err != nil {
			return fmt.Errorf("clearing bun layer: %w", err)
		}

		ctx.Logf("Installing Bun v%s", version)
		installCmd := []string{"bash", "-c", fmt.Sprintf("curl -fsSL https://bun.sh/install | bash -s bun-v%s 2>/dev/null", version)}
		if _, err := ctx.Exec(installCmd, gcp.WithEnv("BUN_INSTALL="+bunLayer.Path), gcp.WithUserAttribution); err != nil {
			return fmt.Errorf("installing bun: %w", err)
		}
		ctx.SetMetadata(bunLayer, bunVersionKey, version)
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(bunLayer, bunVersionKey, version)

	// We need to update the path here to ensure the version we just installed takes precedence.
	if err := ctx.Setenv("PATH", installDir+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}

// detectBunVersion determines the version of bun that should be installed in a Node.js project
// by examining the "engines.bun" and "packageManager" constraints specified in package.json and comparing them against all
// published versions in the NPM registry, if both exist "engines.bun" will take precedence.
// If the package.json does not include "engines.bun" or "packageManager" it
// returns the latest stable version available.
func detectBunVersion(pjs *PackageJSON) (string, error) {
	const bunPackageName = "bun"

	if pjs == nil || (pjs.Engines.Bun == "" && pjs.PackageManager == "") {
		version, err := latestPackageVersion(bunPackageName)
		if err != nil {
			return "", gcp.InternalErrorf("fetching available bun versions: %w", err)
		}
		return version, nil
	}
	var requestedVersion string
	if pjs.Engines.Bun != "" {
		requestedVersion = pjs.Engines.Bun
	} else {
		packageManagerName, packageManagerVersion, err := parsePackageManager(pjs.PackageManager)
		if err != nil {
			return "", err
		}
		if packageManagerName != "bun" {
			return "", gcp.UserErrorf("bun was detected but %s is set in the packageManager package.json field", packageManagerName)
		}
		requestedVersion = packageManagerVersion
	}
	version, err := resolvePackageVersion(bunPackageName, requestedVersion)
	if err != nil {
		return "", gcp.UserErrorf("finding bun version that matched %q: %w", requestedVersion, err)
	}
	return version, nil
}
