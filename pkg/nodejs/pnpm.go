package nodejs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

var (
	// PNPMLock is the name of the pnpm lock file.
	PNPMLock = "pnpm-lock.yaml"
	// pnpmDownloadURL is the template used to generate a pnpm download URL.
	pnpmDownloadURL = "https://github.com/pnpm/pnpm/releases/download/v%s/pnpm-linux-x64"
	// pnpmVersionKey is the metadata key used to store the pnpm version in the pnpn layer.
	pnpmVersionKey = "version"
)

// InstallPNPM installs pnpm in the given layer if it is not already cached.
func InstallPNPM(ctx *gcp.Context, pnpmLayer *libcnb.Layer, pjs *PackageJSON) error {
	layerName := pnpmLayer.Name
	installDir := filepath.Join(pnpmLayer.Path, "bin")
	version, err := detectPNPMVersion(pjs)
	if err != nil {
		return err
	}
	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(pnpmLayer, pnpmVersionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("pnpm cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(pnpmLayer); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install pnpm in layer.
		ctx.Logf("Installing pnpm v%s", version)
		if err := downloadPNPM(ctx, installDir, version); err != nil {
			return gcp.InternalErrorf("downloading pnpm: %w", err)
		}
		fp := filepath.Join(installDir, "pnpm")
		if err := os.Chmod(fp, 0777); err != nil {
			return gcp.InternalErrorf("chmoding %s: %w", fp, err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(pnpmLayer, versionKey, version)
	// We need to update the path here to ensure the version we just installed take precedence over
	// anything pre-installed in the base image.
	if err := ctx.Setenv("PATH", installDir+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}

// downloadPNPM downloads a given version of pnpm into the provided directory.
func downloadPNPM(ctx *gcp.Context, dir, version string) error {
	if err := ctx.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fp := filepath.Join(dir, "pnpm")
	url := fmt.Sprintf(pnpmDownloadURL, version)
	return fetch.File(url, fp)
}

// detectPnpmVersion determines the version of pnpm that should be installed in a Node.js project
// by examining the "engines.pnpm" and "packageManager" constraints specified in package.json and comparing them against all
// published versions in the NPM registry, if both exist "engines.pnpm" will take precedence.
// If the package.json does not include "engines.pnpm" or "packageManager" it
// returns the latest stable version available.
// TODO(b/338411091) create a shared packagejson util library and refactor out a generic detect
// package manager version function.
func detectPNPMVersion(pjs *PackageJSON) (string, error) {
	if pjs == nil || (pjs.Engines.PNPM == "" && pjs.PackageManager == "") {
		version, err := latestPackageVersion("pnpm")
		if err != nil {
			return "", gcp.InternalErrorf("fetching available pnpm versions: %w", err)
		}
		return version, nil
	}
	var requestedVersion string
	if pjs.Engines.PNPM != "" {
		requestedVersion = pjs.Engines.PNPM
	} else {
		packageManagerName, packageManagerVersion, err := parsePackageManager(pjs.PackageManager)
		if err != nil {
			return "", err
		}
		if packageManagerName != "pnpm" {
			return "", gcp.UserErrorf("pnpm was detected but %s is set in the packageManager package.json field.", packageManagerName)
		}
		requestedVersion = packageManagerVersion
	}
	version, err := resolvePackageVersion("pnpm", requestedVersion)
	if err != nil {
		return "", gcp.UserErrorf("finding pnpm version that matched %q: %w", requestedVersion, err)
	}
	return version, nil
}
