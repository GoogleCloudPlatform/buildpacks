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

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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
