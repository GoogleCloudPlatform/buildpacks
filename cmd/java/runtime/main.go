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

// Implements java/runtime buildpack.
// The runtime buildpack installs the JDK.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpack/libbuildpack/buildpackplan"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	javaLayer = "java"
	// TODO: Allow other versions of Java outside of Java11
	javaVersionURL = "https://api.adoptopenjdk.net/v2/info/releases/openjdk11?os=linux&arch=x64&heap_size=normal&openjdk_impl=hotspot&type=jdk"
	versionPrefix  = "jdk-"
)

// metadata represents metadata stored for a runtime layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	runtime.CheckOverride(ctx, "java")

	if !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.java") && !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "**/*.jar") {
		ctx.OptOut("No *.java or *.jar files found.")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	var query, version, archiveURL, releaseURL string
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		query = "&release=" + versionPrefix + url.QueryEscape(v)
		version = v
		ctx.Logf("Using requested runtime version: %s", version)
	} else {
		ctx.Logf("Using latest Java 11 runtime version.")
	}

	releaseURL = javaVersionURL + query
	if code := ctx.HTTPStatus(releaseURL); code != http.StatusOK {
		return gcp.UserErrorf("Runtime version %s does not exist at %s (status %d). You can specify the version with %s.", version, releaseURL, code, env.RuntimeVersion)
	}

	result := ctx.Exec([]string{"curl", "--silent", releaseURL})
	release, err := parseVersionJSON(result.Stdout)
	if err != nil {
		return fmt.Errorf("parsing JSON returned by %s: %w", releaseURL, err)
	}

	version, archiveURL, err = extractRelease(release)
	if err != nil {
		return fmt.Errorf("extracting release returned by %s: %w", releaseURL, err)
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	var meta metadata
	l := ctx.Layer(javaLayer)
	ctx.ReadMetadata(l, &meta)
	if version == meta.Version {
		ctx.CacheHit(javaLayer)
		return nil
	}
	ctx.CacheMiss(javaLayer)
	ctx.ClearLayer(l)

	// Download and install Java in layer.
	ctx.Logf("Installing Java v%s", version)

	command := fmt.Sprintf("curl --fail --show-error --silent --location %s | tar xz --directory=%s --strip-components=1", archiveURL, l.Root)
	ctx.Exec([]string{"bash", "-c", command})

	meta.Version = version
	ctx.WriteMetadata(l, meta, layers.Build, layers.Cache, layers.Launch)

	ctx.AddBuildpackPlan(buildpackplan.Plan{
		Name:    javaLayer,
		Version: version,
	})
	return nil
}

type binary struct {
	BinaryLink   string `json:"binary_link"`
	BinaryType   string `json:"binary_type"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type javaRelease struct {
	Version  string   `json:"release_name"`
	Binaries []binary `json:"binaries"`
}

// parseVersionJSON parses a JSON array of version infomation
func parseVersionJSON(jsonStr string) (javaRelease, error) {
	var releases []javaRelease
	if err := json.Unmarshal([]byte(jsonStr), &releases); err != nil {
		return javaRelease{}, fmt.Errorf("parsing JSON response %q: %v", jsonStr, err)
	}
	if len(releases) == 0 {
		return javaRelease{}, fmt.Errorf("empty list of releases")
	}
	return releases[len(releases)-1], nil
}

// extractRelease returns the version name and archiveURL from a javaRelease.
func extractRelease(release javaRelease) (string, string, error) {
	if len(release.Binaries) == 0 {
		return "", "", fmt.Errorf("no binaries in given release %s", release.Version)
	}

	for _, binary := range release.Binaries {
		if binary.BinaryType == "jdk" && binary.OS == "linux" && binary.Architecture == "x64" {
			return strings.TrimPrefix(release.Version, versionPrefix), binary.BinaryLink, nil
		}
	}

	return "", "", fmt.Errorf("jdk/linux/x64 binary not found in release %s", release.Version)
}
