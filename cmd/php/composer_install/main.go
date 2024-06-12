// Copyright 2020 Google LLC
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

// Implements php/composer-install buildpack.
// The composer-install buildpack installs the composer dependency manager.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fetch"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
)

var (
	composerLayer    = "composer"
	composerJSON     = "composer.json"
	composerSetup    = "composer-setup"
	composerVer      = "2.2.24"
	versionKey       = "version"
	composerSigURL   = "https://composer.github.io/installer.sig"
	composerSetupURL = "https://getcomposer.org/installer"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		// functions-frameworks buildpack expect composer sdk to be installed always.
		return gcp.OptInAlways(), nil
	}
	composerJSONExists, err := ctx.FileExists(composerJSON)
	if err != nil {
		return nil, err
	}
	if !composerJSONExists {
		return gcp.OptOutFileNotFound(composerJSON), nil
	}
	return gcp.OptInFileFound(composerJSON), nil
}

func buildFn(ctx *gcp.Context) error {
	if ver, present := os.LookupEnv(php.ComposerVersion); present {
		composerVer = ver
	}

	l, err := ctx.Layer(composerLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", composerLayer, err)
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(l, versionKey)
	if composerVer == metaVersion {
		ctx.CacheHit(composerLayer)
		ctx.Logf("composer binary cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(composerLayer)
	if err := ctx.ClearLayer(l); err != nil {
		return fmt.Errorf("clearing layer %q: %w", l.Name, err)
	}

	// download the installer
	installer, err := os.CreateTemp(l.Path, fmt.Sprintf("%s-*.php", composerSetup))
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(installer.Name())

	if err := fetch.GetURL(composerSetupURL, installer); err != nil {
		return fmt.Errorf("failed to download composer installer from %s: %w", composerSetupURL, err)
	}

	// verify the installer hash
	var expectedSHABuf bytes.Buffer
	if err := fetch.GetURL(composerSigURL, io.Writer(&expectedSHABuf)); err != nil {
		return fmt.Errorf("failed to fetch the installer signature from %s: %w", composerSigURL, err)
	}
	expectedSHA := expectedSHABuf.String()
	// Disable display_errors to avoid printing warnings to the console when adding the php.ini.
	actualSHACmd := fmt.Sprintf("php -d 'display_errors = Off' -r \"echo hash_file('sha384', '%s');\"", installer.Name())
	result, err := ctx.Exec([]string{"bash", "-c", actualSHACmd})
	if err != nil {
		return err
	}
	actualSHA := result.Stdout
	if actualSHA != expectedSHA {
		return fmt.Errorf("invalid composer installer found at %q: checksum for composer installer, %q, does not match expected checksum of %q", composerSetupURL, actualSHA, expectedSHA)
	}

	// run the installer
	ctx.Logf("installing Composer v%s", composerVer)
	clBin := filepath.Join(l.Path, "bin")
	if err := ctx.MkdirAll(clBin, 0755); err != nil {
		return fmt.Errorf("creating bin folder: %w", err)
	}
	installCmd := fmt.Sprintf("php %s --install-dir %s --filename composer --version %s", installer.Name(), clBin, composerVer)
	if _, err := ctx.Exec([]string{"bash", "-c", installCmd}); err != nil {
		return err
	}

	ctx.SetMetadata(l, versionKey, composerVer)
	return nil
}
