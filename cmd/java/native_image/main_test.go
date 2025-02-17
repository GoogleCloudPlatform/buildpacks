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

package main

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb/v2"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "always opting in",
			want: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestFindSpringBootPlugin(t *testing.T) {
	testCases := []struct {
		name         string
		mavenProject *java.MavenProject
		want         bool
	}{
		{
			name: "spring boot plugin defined",
			mavenProject: &java.MavenProject{
				Plugins: []java.MavenPlugin{
					{GroupID: "org.springframework.boot", ArtifactID: "spring-boot-maven-plugin"},
				},
			},
			want: true,
		},
		{
			name: "only group ID matches",
			mavenProject: &java.MavenProject{
				Plugins: []java.MavenPlugin{
					{GroupID: "org.springframework.boot", ArtifactID: "different-plugin"},
				},
			},
			want: false,
		},
		{
			name: "only artifact ID matches",
			mavenProject: &java.MavenProject{
				Plugins: []java.MavenPlugin{
					{GroupID: "com.example.foo", ArtifactID: "spring-boot-maven-plugin"},
				},
			},
			want: false,
		},
		{
			name: "spring boot first and other plugins",
			mavenProject: &java.MavenProject{
				Plugins: []java.MavenPlugin{
					{GroupID: "org.springframework.boot", ArtifactID: "spring-boot-maven-plugin"},
					{GroupID: "com.example.foo", ArtifactID: "bar-plugin"},
				},
			},
			want: true,
		},
		{
			name: "multiple plugins",
			mavenProject: &java.MavenProject{
				Plugins: []java.MavenPlugin{
					{GroupID: "com.example.foo", ArtifactID: "bar-plugin"},
					{GroupID: "org.springframework.boot", ArtifactID: "spring-boot-maven-plugin"},
					{GroupID: "com.google.guava", ArtifactID: "guava"},
				},
			},
			want: true,
		},
		{
			name:         "spring boot plugin undefined",
			mavenProject: &java.MavenProject{},
			want:         false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()
			got := springBootPluginDefined(ctx, tc.mavenProject)
			if got != tc.want {
				t.Errorf("findSpringBootPlugin()=%v want %v", got, tc.want)
			}
		})
	}
}

func TestGetClasspathAndMainFromSpringBoot(t *testing.T) {
	testCases := []struct {
		name          string
		setupJar      bool
		manifest      string
		filesInJar    []string
		wantClasspath *regexp.Regexp
		wantMain      string
	}{
		{
			name:          "normal spring boot",
			setupJar:      true,
			manifest:      "Main-Class: com.example.Main\nStart-Class: com.example.Start",
			wantClasspath: regexp.MustCompile(".+/exploded-jar:.+/exploded-jar/BOOT-INF/classes:.+/exploded-jar/BOOT-INF/lib/\\*"),
			wantMain:      "com.example.Start",
		},
		{
			name:          "no Start-Class in manifest",
			setupJar:      true,
			manifest:      "Main-Class: com.example.Main",
			wantClasspath: regexp.MustCompile("^$"),
			wantMain:      "",
		},
		{
			name:          "no executable JAR",
			setupJar:      false,
			manifest:      "Main-Class: com.example.Main\nStart-Class: com.example.Start",
			wantClasspath: regexp.MustCompile("^$"),
			wantMain:      "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jarDir := "."
			if tc.setupJar {
				jar := setupTestJar(t, []byte(tc.manifest))
				jarDir = path.Dir(jar)
			}
			ctx := gcp.NewContext(gcp.WithApplicationRoot(jarDir), gcp.WithBuildContext(libcnb.BuildContext{Layers: libcnb.Layers{Path: jarDir}}))

			classpath, main, err := classpathAndMainFromSpringBoot(ctx)
			if err != nil {
				t.Errorf("classpathAndMainFromSpringBoot() failed: %v", err)
			}
			if tc.wantClasspath.FindStringIndex(classpath) == nil {
				t.Errorf("wrong classpath: got %q, want %q", classpath, tc.wantClasspath)
			}
			if main != tc.wantMain {
				t.Errorf("wrong main: got %q, want %q", main, tc.wantMain)
			}
		})
	}
}

func setupTestJar(t *testing.T, mfContent []byte) string {
	t.Helper()
	var buff bytes.Buffer
	w := zip.NewWriter(&buff)
	defer w.Close()
	f, err := w.Create(filepath.Join("META-INF", "MANIFEST.MF"))
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	for i := 0; i < len(mfContent); {
		n, err := f.Write(mfContent)
		if err != nil {
			t.Fatalf("failed to write bytes: %v", err)
		}
		i += n
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	jarPath := filepath.Join(t.TempDir(), "test.jar")
	if err := ioutil.WriteFile(jarPath, buff.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write to file %s: %v", jarPath, err)
	}
	return jarPath
}
