// Copyright 2025 Google LLC
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
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

var (
	// BunLock is the name of the bun lock file (text format).
	BunLock = "bun.lock"
	// BunLockB is the name of the bun lock file (binary format).
	BunLockB = "bun.lockb"
	// bunDownloadURL is the template used to generate a bun download URL.
	bunDownloadURL = "https://github.com/oven-sh/bun/releases/download/bun-v%s/bun-linux-x64.zip"
	// bunVersionKey is the metadata key used to store the bun version in the bun layer.
	bunVersionKey = "version"
)

// InstallBun installs bun in the given layer if it is not already cached.
func InstallBun(ctx *gcp.Context, bunLayer *libcnb.Layer, pjs *PackageJSON) error {
	layerName := bunLayer.Name
	installDir := filepath.Join(bunLayer.Path, "bin")
	version, err := detectBunVersion(pjs)
	if err != nil {
		return err
	}
	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(bunLayer, bunVersionKey)
	if version == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("bun cache hit: %q, %q, skipping installation.", version, metaVersion)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(bunLayer); err != nil {
			return fmt.Errorf("clearing layer %q: %w", layerName, err)
		}
		// Download and install bun in layer.
		ctx.Logf("Installing bun v%s", version)
		if err := downloadBun(ctx, installDir, version); err != nil {
			return gcp.InternalErrorf("downloading bun: %w", err)
		}
	}

	// Store layer flags and metadata.
	ctx.SetMetadata(bunLayer, bunVersionKey, version)
	// We need to update the path here to ensure the version we just installed take precedence over
	// anything pre-installed in the base image.
	if err := ctx.Setenv("PATH", installDir+":"+os.Getenv("PATH")); err != nil {
		return err
	}
	return nil
}

// downloadBun downloads a given version of bun into the provided directory.
func downloadBun(ctx *gcp.Context, dir, version string) error {
	if err := ctx.MkdirAll(dir, 0755); err != nil {
		return err
	}
	url := fmt.Sprintf(bunDownloadURL, version)
	// Bun is distributed as a zip file, extract it to the directory
	archivePath := filepath.Join(dir, "bun.zip")
	if err := fetch.File(url, archivePath); err != nil {
		return err
	}
	// Extract the zip file
	if _, err := ctx.Exec([]string{"unzip", "-o", archivePath, "-d", dir}, gcp.WithWorkDir(dir)); err != nil {
		return fmt.Errorf("extracting bun archive: %w", err)
	}
	// Move bun binary from extracted folder to bin directory
	extractedBunPath := filepath.Join(dir, "bun-linux-x64", "bun")
	targetBunPath := filepath.Join(dir, "bun")
	if err := os.Rename(extractedBunPath, targetBunPath); err != nil {
		return fmt.Errorf("moving bun binary: %w", err)
	}
	// Make executable
	if err := os.Chmod(targetBunPath, 0755); err != nil {
		return gcp.InternalErrorf("chmoding %s: %w", targetBunPath, err)
	}
	// Clean up
	if err := os.RemoveAll(filepath.Join(dir, "bun-linux-x64")); err != nil {
		return fmt.Errorf("cleaning up extracted directory: %w", err)
	}
	if err := os.Remove(archivePath); err != nil {
		return fmt.Errorf("cleaning up archive: %w", err)
	}
	return nil
}

// detectBunVersion determines the version of bun that should be installed in a Node.js project
// by examining the "engines.bun" and "packageManager" constraints specified in package.json and comparing them against all
// published versions in the NPM registry, if both exist "engines.bun" will take precedence.
// If the package.json does not include "engines.bun" or "packageManager" it
// returns the latest stable version available.
func detectBunVersion(pjs *PackageJSON) (string, error) {
	if pjs == nil || (pjs.Engines.Bun == "" && pjs.PackageManager == "") {
		version, err := latestPackageVersion("bun")
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
			return "", gcp.UserErrorf("bun was detected but %s is set in the packageManager package.json field.", packageManagerName)
		}
		requestedVersion = packageManagerVersion
	}
	version, err := resolvePackageVersion("bun", requestedVersion)
	if err != nil {
		return "", gcp.UserErrorf("finding bun version that matched %q: %w", requestedVersion, err)
	}
	return version, nil
}
