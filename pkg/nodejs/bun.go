package nodejs

import (
	"fmt"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb/v2"
)

// BunInstallerCapability is the capability key for the BunInstaller.
const BunInstallerCapability = "nodejs.BunInstaller"

// bunInstaller is an interface for installing bun.
type bunInstaller interface {
	InstallBun(ctx *gcp.Context, bunLayer *libcnb.Layer, pjs *PackageJSON) error
}

var (
	// BunLock is the name of the Bun lock file.
	BunLock = "bun.lock"
	// BunLockb is the name of the Bun lock file in binary format.
	BunLockb = "bun.lockb"
	// bunVersionKey is the metadata key used to store the Bun version in the bun layer.
	bunVersionKey = "version"
)

// InstallBun installs Bun in the given layer using the curl installer.
func InstallBun(ctx *gcp.Context, bunLayer *libcnb.Layer, pjs *PackageJSON) error {
	if ctx.IsDisabled(BunInstallerCapability) {
		ctx.Logf("BunInstaller capability is disabled. Skipping installation.")
		return nil
	}

	if cap := ctx.Capability(BunInstallerCapability); cap != nil {
		i, ok := cap.(bunInstaller)
		if !ok {
			return gcp.InternalErrorf("capability %q must implement BunInstaller", BunInstallerCapability)
		}
		return i.InstallBun(ctx, bunLayer, pjs)
	}

	installDir := filepath.Join(bunLayer.Path, "bin")
	version, err := detectBunVersion(pjs)
	if err != nil {
		return err
	}

	cached, err := runtime.InstallTarballIfNotCached(ctx, runtime.Bun, version, bunLayer)
	// TODO(b/520284867): Remove this fallback block once things work fine with the AR approach.
	if err != nil {
		ctx.Logf("Failed to download Bun v%s tarball: %v. Falling back to script download.", version, err)
		if err := ctx.ClearLayer(bunLayer); err != nil {
			return fmt.Errorf("clearing bun layer: %w", err)
		}
		ctx.Logf("Installing Bun v%s via script", version)
		installCmd := []string{"bash", "-c", fmt.Sprintf("curl -fsSL https://bun.sh/install | bash -s bun-v%s 2>/dev/null", version)}
		if _, err := ctx.Exec(installCmd, gcp.WithEnv("BUN_INSTALL="+bunLayer.Path), gcp.WithUserAttribution); err != nil {
			return fmt.Errorf("installing bun: %w", err)
		}
		ctx.SetMetadata(bunLayer, bunVersionKey, version)
		ctx.SetMetadata(bunLayer, "stack", ctx.StackID())
	} else if !cached {
		ctx.Logf("Successfully installed Bun v%s from tarball.", version)
	}

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
