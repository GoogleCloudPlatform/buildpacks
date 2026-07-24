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

package java

import (
	"embed"
	"reflect"
	"testing"
)

//go:embed testdata/*
var testData embed.FS

func TestParseValidPom(t *testing.T) {
	tests := []struct {
		path string
		want MavenProject
	}{
		{
			path: "testdata/simple_project.xml",
			want: MavenProject{
				Plugins: []MavenPlugin{
					{
						GroupID:    "org.apache.maven.plugins",
						ArtifactID: "maven-jar-plugin",
					},
				},
				Profiles: []MavenProfile{
					{
						ID: "native",
						Plugins: []MavenPlugin{
							{
								GroupID:    "org.graalvm.nativeimage",
								ArtifactID: "native-image-maven-plugin",
								Configuration: MavenPluginConfiguration{
									MainClass: "com.example.Driver",
									BuildArgs: "--no-server",
								},
							},
						},
					},
				},
				ArtifactID: "firestore-sample",
				Version:    "",
				Dependencies: []MavenDependency{
					{
						GroupID:    "com.google.cloud",
						ArtifactID: "google-cloud-graalvm-support",
					},
					{
						GroupID:    "com.google.cloud",
						ArtifactID: "google-cloud-core",
					},
					{
						GroupID:    "com.google.cloud",
						ArtifactID: "google-cloud-firestore",
					},
				},
			},
		},
		{
			path: "testdata/empty_project.xml",
			want: MavenProject{
				Plugins:  nil,
				Profiles: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			pomFile, err := testData.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("Unable to find pom file %s, %v", tc.path, err)
			}

			got, err := ParsePomFile(pomFile)
			if err != nil {
				t.Fatalf("ParsePomFile failed to parse pom.xml: %v", err)
			}

			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("ParsePomFile\ngot %#v\nwant %#v", *got, tc.want)
			}
		})
	}
}

func TestParseInvalidPom(t *testing.T) {
	tests := []string{
		"testdata/invalid_project.xml",
		"testdata/empty_file.xml",
	}

	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			pomFile, err := testData.ReadFile(tc)
			if err != nil {
				t.Fatalf("Unable to find pom file %s, %v", tc, err)
			}

			_, err = ParsePomFile(pomFile)
			if err == nil { // if NO error
				t.Errorf("ParsePomFile succeeded for invalid pom: %s, want error", tc)
			}
		})
	}
}
