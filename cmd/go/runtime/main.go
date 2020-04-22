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

// Implements go/runtime buildpack.
// The runtime buildpack installs the Go toolchain.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	// goVersionURL is a URL to a JSON file that contains the latest Go version names.
	goVersionURL = "https://golang.org/dl/?mode=json"
	goURL        = "https://dl.google.com/go/go%s.linux-amd64.tar.gz"
	goLayer      = "go"
)

// metadata represents metadata stored for a runtime layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	runtime.CheckOverride(ctx, "go")

	if !ctx.HasAtLeastOne("*.go") {
		ctx.OptOut("No *.go files found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := runtimeVersion(ctx)
	if err != nil {
		return err
	}
	grl := ctx.Layer(goLayer)
	// Check metadata layer to see if correct version of Go is already installed.
	var meta metadata
	ctx.ReadMetadata(grl, &meta)
	if version == meta.Version {
		ctx.CacheHit(goLayer)
	} else {
		ctx.CacheMiss(goLayer)
		ctx.ClearLayer(grl)

		archiveURL := fmt.Sprintf(goURL, version)
		if code := ctx.HTTPStatus(archiveURL); code != http.StatusOK {
			return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, archiveURL, code, env.RuntimeVersion)
		}

		// Download and install Go in layer.
		ctx.Logf("Installing Go v%s", version)
		command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, grl.Root)
		ctx.Exec([]string{"bash", "-c", command})

		meta.Version = version
	}

	// Write the layer information.
	lf := []layers.Flag{layers.Build, layers.Cache}
	if devmode.Enabled(ctx) {
		lf = append(lf, layers.Launch)
	}
	ctx.WriteMetadata(grl, meta, lf...)

	return nil
}

func runtimeVersion(ctx *gcp.Context) (string, error) {
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	if version := golang.GoModVersion(ctx); version != "" {
		ctx.Logf("Using runtime version from go.mod: %s", version)
		return version, nil
	}
	version, err := latestGoVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("getting latest version: %w", err)
	}
	ctx.Logf("Using latest runtime version: %s", version)
	return version, nil
}

type goReleases []struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// latestGoVersion returns the latest version of Go
func latestGoVersion(ctx *gcp.Context) (string, error) {
	result := ctx.Exec([]string{"curl", "--silent", goVersionURL})
	return parseVersionJSON(result.Stdout)
}

func parseVersionJSON(jsonStr string) (string, error) {
	releases := goReleases{}
	if err := json.Unmarshal([]byte(jsonStr), &releases); err != nil {
		return "", fmt.Errorf("parsing JSON response from URL %q: %v", goVersionURL, err)
	}

	for _, release := range releases {
		if !release.Stable {
			continue
		}
		if v := strings.TrimPrefix(release.Version, "go"); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("parsing latest stable version from %q", goVersionURL)
}
