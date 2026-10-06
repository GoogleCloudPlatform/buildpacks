// Copyright 2021 Google LLC
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

package java

import (
	"encoding/xml"

	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/tooling"
)

const (
	defaultMavenVersion = "3.9.11"
	mavenLayer          = "maven"
	versionKey          = "version"

	// MavenInstallerCapability is the capability key for the maker Maven builder.
	MavenInstallerCapability = "java.MavenInstaller"
)

var (
	mavenURLs = []string{
		// The CDN URL is incredibly fast (> 10x faster) but it only has latest version(s). Fall-back to the archive URL when the current version is no longer in the CDN
		"https://dlcdn.apache.org/maven/maven-3/%[1]s/binaries/apache-maven-%[1]s-bin.tar.gz",
		"https://archive.apache.org/dist/maven/maven-3/%[1]s/binaries/apache-maven-%[1]s-bin.tar.gz",
	}
	validStatuses = []string{
		"HTTP/1.1 200 OK",
		"HTTP/2 200",
	}
)

// MavenProject is the root struct that contains the unmarshalled pom.xml.
type MavenProject struct {
	Plugins      []MavenPlugin     `xml:"build>plugins>plugin"`
	Profiles     []MavenProfile    `xml:"profiles>profile"`
	ArtifactID   string            `xml:"artifactId"`
	Version      string            `xml:"version"`
	Dependencies []MavenDependency `xml:"dependencies>dependency"`
}

// MavenProfile describes a profile defined in the pom.xml.
type MavenProfile struct {
	ID           string            `xml:"id"`
	Plugins      []MavenPlugin     `xml:"build>plugins>plugin"`
	Dependencies []MavenDependency `xml:"dependencies>dependency"`
}

// MavenPlugin describes plugins defined in the pom.xml.
type MavenPlugin struct {
	GroupID       string                   `xml:"groupId"`
	ArtifactID    string                   `xml:"artifactId"`
	Configuration MavenPluginConfiguration `xml:"configuration"`
}

// MavenDependency describes a dependency in the pom.xml.
type MavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
}

// MavenPluginConfiguration describes plugin settings that are parsed from the pom.xml.
type MavenPluginConfiguration struct {
	MainClass string `xml:"mainClass"`
	BuildArgs string `xml:"buildArgs"`
}

// ParsePomFile unmarshals the provided pom.xml into a MavenProject.
func ParsePomFile(pomFile []byte) (*MavenProject, error) {
	var proj MavenProject
	if err := xml.Unmarshal(pomFile, &proj); err != nil {
		return nil, gcp.UserErrorf("parsing pom.xml: %v", err)
	}

	return &proj, nil
}

// InstallMaven installs Maven and returns the path of the mvn binary
func InstallMaven(ctx *gcp.Context) (string, error) {
	mvnl, err := ctx.Layer(mavenLayer, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return "", fmt.Errorf("creating %v layer: %w", mavenLayer, err)
	}

	// Check the metadata in the cache layer to determine if we need to proceed.
	metaVersion := ctx.GetMetadata(mvnl, versionKey)
	mavenVersion := resolveMavenVersion(ctx)
	if mavenVersion == metaVersion {
		ctx.CacheHit(mavenLayer)
		ctx.Logf("Maven cache hit, skipping installation.")
		return filepath.Join(mvnl.Path, "bin", "mvn"), nil
	}
	ctx.CacheMiss(mavenLayer)
	if err := ctx.ClearLayer(mvnl); err != nil {
		return "", fmt.Errorf("clearing layer %q: %w", mvnl.Name, err)
	}

	// Download and install maven in layer.
	ctx.Logf("Installing Maven v%s", mavenVersion)
	archiveURL, err := resolveMavenURL(ctx, mavenVersion)
	if err != nil {
		return "", fmt.Errorf("failed to get Maven: %w", err)
	}
	command := fmt.Sprintf("curl --fail --show-error --silent --location --retry 3 %s | tar xz --directory %s --strip-components=1", archiveURL, mvnl.Path)
	if _, err := ctx.Exec([]string{"bash", "-c", command}); err != nil {
		return "", err
	}
	ctx.Logf("Downloading Maven from %s", archiveURL)

	ctx.SetMetadata(mvnl, versionKey, mavenVersion)
	return filepath.Join(mvnl.Path, "bin", "mvn"), nil
}

func resolveMavenVersion(ctx *gcp.Context) string {
	latestVersion, err := tooling.ResolveToolVersion("java", "maven", os.Getenv(env.RuntimeVersion), runtime.OSForStack(ctx))
	if err != nil || latestVersion == "" {
		ctx.Warnf("Could not resolve pinned maven version, falling back to latest: %v, %v", defaultMavenVersion, err)
		latestVersion = defaultMavenVersion
	} else {
		ctx.Logf("Using Maven version %s from tooling.ResolveToolVersion", latestVersion)
	}
	return latestVersion
}

func resolveMavenURL(ctx *gcp.Context, mavenVersion string) (string, error) {
	for _, url := range mavenURLs {
		archiveURL := fmt.Sprintf(url, mavenVersion)
		curlHead := fmt.Sprintf("curl --head --fail --silent --location %s", archiveURL)
		result, err := ctx.Exec([]string{"bash", "-c", curlHead})
		if err == nil {
			for _, status := range validStatuses {
				if strings.Contains(result.Stdout, status) {
					return archiveURL, nil
				}
			}
		}
	}
	return "", gcp.InternalErrorf("Maven version %s does not exist at any of the URLs %v (status not 200).", mavenVersion, mavenURLs)
}
