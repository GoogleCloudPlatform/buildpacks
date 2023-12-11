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

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestGetAssemblyName(t *testing.T) {
	tcs := []struct {
		name string
		want string
		err  bool
		data string
	}{
		{
			name: "no AssemblyName fields",
			err:  true,
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

	</Project>`,
		},
		{
			name: "one AssemblyName field",
			want: "MyApp",
			err:  false,
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

		<PropertyGroup>
			<AssemblyName>MyApp</AssemblyName>
		</PropertyGroup>

	</Project>`,
		},
		{
			name: "two AssemblyName fields",
			want: "",
			err:  true,
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

		<PropertyGroup>
			<AssemblyName>MyApp</AssemblyName>
		</PropertyGroup>

		<PropertyGroup>
			<AssemblyName>Oopsie</AssemblyName>
		</PropertyGroup>

	</Project>`,
		},
		{
			name: "malformed xml",
			want: "",
			err:  true,
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

		<PropertyGroup>

	</Project>`,
		},
	}
	for _, tc := range tcs {
		ctx := gcp.NewContext()
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dotnettest")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			filename := filepath.Join(tmpDir, "app.csproj")
			if err = os.WriteFile(filename, []byte(tc.data), 0644); err != nil {
				t.Fatalf("writing project file: %v", err)
			}

			v, err := getAssemblyName(ctx, filename)
			if err != nil {
				if !tc.err {
					t.Errorf("got no error, want an error")
				}
				return
			}
			if v != tc.want {
				t.Errorf("got %s, want %s", v, tc.want)
			}
		})
	}
}

func TestGetEntrypoint(t *testing.T) {
	tcs := []struct {
		name string
		exe  string
		proj string
		data string
		want string
	}{
		{
			name: "dll from project file",
			exe:  "myapp.dll",
			proj: "myapp.proj",
			want: "cd {{.Tmp}} && exec dotnet myapp.dll",
		},
		{
			name: "dll from project file with dots",
			exe:  "my.app.dll",
			proj: "my.app.proj",
			want: "cd {{.Tmp}} && exec dotnet my.app.dll",
		},
		{
			name: "exe from assembly name",
			exe:  "customapp.dll",
			proj: "myapp.proj",
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

		<PropertyGroup>
			<AssemblyName>customapp</AssemblyName>
		</PropertyGroup>

	</Project>`,
			want: "cd {{.Tmp}} && exec dotnet customapp.dll",
		},
		{
			name: "dll from assembly name",
			exe:  "customapp.dll",
			proj: "myapp.proj",
			data: `<Project Sdk="Microsoft.NET.Sdk.Web">

		<PropertyGroup>
			<AssemblyName>customapp</AssemblyName>
		</PropertyGroup>

	</Project>`,
			want: "cd {{.Tmp}} && exec dotnet customapp.dll",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gcp.NewContext()

			tmpDir, err := os.MkdirTemp("", "dotnettest")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					t.Fatalf("removing temp dir: %v", err)
				}
			}()

			// Write the expected exe file.
			exe := filepath.Join(tmpDir, tc.exe)
			if err = os.WriteFile(exe, []byte(""), 0644); err != nil {
				t.Fatalf("writing exe file: %v", err)
			}

			// Write the project file.
			proj := filepath.Join(tmpDir, tc.proj)
			if err = os.WriteFile(proj, []byte(tc.data), 0644); err != nil {
				t.Fatalf("writing proj file: %v", err)
			}

			ep, err := getEntrypoint(ctx, tmpDir, proj)
			if err != nil {
				t.Fatalf("getting entrypoint: %v", err)
			}

			tmpl, err := template.New("want").Parse(tc.want)
			if err != nil {
				t.Fatalf("executing template: %v", err)
			}

			var buf bytes.Buffer
			if err = tmpl.Execute(&buf, struct{ Tmp string }{tmpDir}); err != nil {
				t.Fatalf("executing template: %v", err)
			}

			if want := buf.String(); ep != want {
				t.Errorf("got %s, want %s", ep, want)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		want  int
	}{
		{
			name: "csproj",
			files: map[string]string{
				"Program.cs": "",
				"app.csproj": "",
			},
			want: 0,
		},
		{
			name: "fsproj",
			files: map[string]string{
				"Program.fs": "",
				"app.fsproj": "",
			},
			want: 0,
		}, {
			name: "vbproj",
			files: map[string]string{
				"Program.vb": "",
				"app.vbproj": "",
			},
			want: 0,
		},
		{
			name: "with build env",
			files: map[string]string{
				"Program.cs": "",
			},
			env:  []string{"GOOGLE_BUILDABLE=myapp"},
			want: 0,
		},
		{
			name: "with project file and build env",
			files: map[string]string{
				"Program.cs": "",
				"app.csproj": "",
			},
			want: 0,
		},
		{
			name: "unsupported .pyproj",
			files: map[string]string{
				".pyproj": "",
			},
			want: 100,
		},
		{
			name: "unsupported partly matching",
			files: map[string]string{
				"Program.cs":   "",
				"app.mycsproj": "",
			},
			want: 100,
		},
		{
			name: "without project file or build env",
			files: map[string]string{
				"Program.cs": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetect(t, detectFn, tc.name, tc.files, tc.env, tc.want)
		})
	}
}

func TestDeleteFolder(t *testing.T) {
	testCases := []struct {
		name         string
		toDelete     string
		createFolder string
		createFiles  []string
		want         bool
	}{
		{
			name:     "target doesn't exist",
			toDelete: "bin",
			want:     false,
		},
		{
			name:        "bin file",
			toDelete:    "bin",
			createFiles: []string{"bin"},
			want:        true,
		},
		{
			name:         "empty folder",
			toDelete:     "bin",
			createFolder: "bin",
			want:         true,
		},
		{
			name:         "non-empty folder",
			toDelete:     "bin",
			createFolder: "bin",
			createFiles:  []string{"bin/a", "bin/b"},
			want:         true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			if tc.createFolder != "" {
				if err := os.MkdirAll(filepath.Join(dir, tc.createFolder), os.ModePerm); err != nil {
					t.Fatalf("error making %v dir: %v", tc.createFolder, err)
				}
			}

			for _, f := range tc.createFiles {
				if _, err := os.Create(filepath.Join(dir, f)); err != nil {
					t.Fatalf("error creating %v: %v", f, err)
				}
			}

			deleted, err := deleteFolder(gcp.NewContext(gcp.WithApplicationRoot(dir)), filepath.Join(dir, tc.toDelete))
			if err != nil {
				t.Fatalf("an error occurred, but none was expected: %v", err)
			}
			if tc.want != deleted {
				t.Errorf("got %v, want %v", deleted, tc.want)
			}
		})
	}
}
