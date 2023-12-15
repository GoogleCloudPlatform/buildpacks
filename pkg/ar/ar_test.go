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

package ar

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestGeneratePythonConfig(t *testing.T) {
	testCases := []struct {
		name       string
		fileExists bool
		tokenError error
		wantConfig string
	}{
		{
			name:       ".netrc already exists",
			fileExists: true,
			wantConfig: "",
		},
		{
			name:       "credential error",
			fileExists: false,
			tokenError: fmt.Errorf("Error fetching token"),
			wantConfig: "",
		},
		{
			name:       ".netrc created",
			fileExists: false,
			wantConfig: fmt.Sprint(`
machine asia-python.pkg.dev login oauth2accesstoken password token
machine asia-east1-python.pkg.dev login oauth2accesstoken password token
machine asia-east2-python.pkg.dev login oauth2accesstoken password token
machine asia-northeast1-python.pkg.dev login oauth2accesstoken password token
machine asia-northeast2-python.pkg.dev login oauth2accesstoken password token
machine asia-northeast3-python.pkg.dev login oauth2accesstoken password token
machine asia-south1-python.pkg.dev login oauth2accesstoken password token
machine asia-south2-python.pkg.dev login oauth2accesstoken password token
machine asia-southeast1-python.pkg.dev login oauth2accesstoken password token
machine asia-southeast2-python.pkg.dev login oauth2accesstoken password token
machine australia-southeast1-python.pkg.dev login oauth2accesstoken password token
machine australia-southeast2-python.pkg.dev login oauth2accesstoken password token
machine europe-python.pkg.dev login oauth2accesstoken password token
machine europe-central2-python.pkg.dev login oauth2accesstoken password token
machine europe-north1-python.pkg.dev login oauth2accesstoken password token
machine europe-southwest1-python.pkg.dev login oauth2accesstoken password token
machine europe-west1-python.pkg.dev login oauth2accesstoken password token
machine europe-west2-python.pkg.dev login oauth2accesstoken password token
machine europe-west3-python.pkg.dev login oauth2accesstoken password token
machine europe-west4-python.pkg.dev login oauth2accesstoken password token
machine europe-west5-python.pkg.dev login oauth2accesstoken password token
machine europe-west6-python.pkg.dev login oauth2accesstoken password token
machine europe-west8-python.pkg.dev login oauth2accesstoken password token
machine europe-west9-python.pkg.dev login oauth2accesstoken password token
machine europe-west10-python.pkg.dev login oauth2accesstoken password token
machine europe-west12-python.pkg.dev login oauth2accesstoken password token
machine me-central1-python.pkg.dev login oauth2accesstoken password token
machine me-central2-python.pkg.dev login oauth2accesstoken password token
machine me-west1-python.pkg.dev login oauth2accesstoken password token
machine northamerica-northeast1-python.pkg.dev login oauth2accesstoken password token
machine northamerica-northeast2-python.pkg.dev login oauth2accesstoken password token
machine southamerica-east1-python.pkg.dev login oauth2accesstoken password token
machine southamerica-west1-python.pkg.dev login oauth2accesstoken password token
machine us-central1-python.pkg.dev login oauth2accesstoken password token
machine us-python.pkg.dev login oauth2accesstoken password token
machine us-east1-python.pkg.dev login oauth2accesstoken password token
machine us-east4-python.pkg.dev login oauth2accesstoken password token
machine us-east5-python.pkg.dev login oauth2accesstoken password token
machine us-south1-python.pkg.dev login oauth2accesstoken password token
machine us-west1-python.pkg.dev login oauth2accesstoken password token
machine us-west2-python.pkg.dev login oauth2accesstoken password token
machine us-west3-python.pkg.dev login oauth2accesstoken password token
machine us-west4-python.pkg.dev login oauth2accesstoken password token
machine us-west8-python.pkg.dev login oauth2accesstoken password token
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub out the logic for fetching Application Default Credentials
			origFindDefaultCredentials := findDefaultCredentials
			findDefaultCredentials = func() (string, error) {
				return "token", tc.tokenError
			}
			defer func() {
				findDefaultCredentials = origFindDefaultCredentials
			}()

			tempHome := t.TempDir()
			t.Setenv("HOME", tempHome)

			ctx := gcp.NewContext()

			filepath := filepath.Join(tempHome, ".netrc")
			if tc.fileExists {
				f, err := os.Create(filepath)
				if err != nil {
					t.Fatalf("error creating %s: %v", filepath, err)
				}
				f.Close()
			}

			if err := GeneratePythonConfig(ctx); err != nil {
				t.Fatalf("Generating config: %v", err)
			}

			config, err := os.ReadFile(filepath)
			if err != nil && tc.wantConfig != "" {
				t.Fatalf("Reading file %s: %v", filepath, err)
			}

			if diff := cmp.Diff(tc.wantConfig, string(config)); diff != "" {
				t.Errorf("unexpected config (+got, -want):\n %v", diff)
			}
		})
	}
}

func TestGenerateNPMConfig(t *testing.T) {
	t.Cleanup(buildermetrics.Reset)
	testCases := []struct {
		name         string
		fileExists   bool
		tokenError   error
		projectNpmrc string
		wantConfig   string
	}{
		{
			name:       "user .npmrc already exists",
			fileExists: true,
		},
		{
			name:       "credential error",
			tokenError: fmt.Errorf("Error fetching token"),
		},
		{
			name: "project .npmrc with npmjs.org config",
			projectNpmrc: fmt.Sprint(`
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
`),
		},
		{
			name: "project .npmrc with AR repo",
			projectNpmrc: fmt.Sprint(`
registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
`),
		},
		{
			name: "project .npmrc with scoped AR repo",
			projectNpmrc: fmt.Sprint(`
@myscope:registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
`),
		},
		{
			name: "project .npmrc with multiple repos",
			projectNpmrc: fmt.Sprint(`
registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
@myscope:registry=https://us-central1-npm.pkg.dev/my-other-project/my-other-repo/
//us-central1-npm.pkg.dev/my-other-project/my-other-repo/:always-auth=true
registry=https://my-site/my-organization/_packaging/my-project/npm/registry/
always-auth=true
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
//us-central1-npm.pkg.dev/my-other-project/my-other-repo/:_authToken=token
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub out the logic for fetching Application Default Credentials
			origFindDefaultCredentials := findDefaultCredentials
			findDefaultCredentials = func() (string, error) {
				return "token", tc.tokenError
			}
			defer func() {
				findDefaultCredentials = origFindDefaultCredentials
			}()

			// set up the application root dir
			tempRoot := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))
			if tc.projectNpmrc != "" {
				filepath := filepath.Join(tempRoot, ".npmrc")
				os.WriteFile(filepath, []byte(tc.projectNpmrc), 0664)
			}

			// set up the $HOME dir
			t.Setenv("HOME", t.TempDir())
			filepath := filepath.Join(ctx.HomeDir(), ".npmrc")
			if tc.fileExists {
				os.WriteFile(filepath, []byte{}, 0664)
			}

			if err := GenerateNPMConfig(ctx); err != nil {
				t.Fatalf("Error generating config: %v", err)
			}

			config, err := os.ReadFile(filepath)
			if err != nil && tc.wantConfig != "" {
				t.Fatalf("Error reading file %s: %v", filepath, err)
			}

			if diff := cmp.Diff(tc.wantConfig, string(config)); diff != "" {
				t.Errorf("unexpected config (+got, -want):\n %v", diff)
			}
		})
	}
}

func TestGenerateNPMConfigMetrics(t *testing.T) {
	t.Cleanup(buildermetrics.Reset)
	successfulCredGens := int64(0)
	testCases := []struct {
		name         string
		fileExists   bool
		tokenError   error
		projectNpmrc string
		wantConfig   string
	}{
		{
			name: "project .npmrc with AR repo",
			projectNpmrc: fmt.Sprint(`
registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
`),
		},
		{
			name: "project .npmrc with scoped AR repo",
			projectNpmrc: fmt.Sprint(`
@myscope:registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
`),
		},
		{
			name: "project .npmrc with multiple repos",
			projectNpmrc: fmt.Sprint(`
registry=https://us-west1-npm.pkg.dev/my-project/my-repo/
//us-west1-npm.pkg.dev/my-project/my-repo/:always-auth=true
@myscope:registry=https://us-central1-npm.pkg.dev/my-other-project/my-other-repo/
//us-central1-npm.pkg.dev/my-other-project/my-other-repo/:always-auth=true
registry=https://my-site/my-organization/_packaging/my-project/npm/registry/
always-auth=true
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
`),
			wantConfig: fmt.Sprint(`
//us-west1-npm.pkg.dev/my-project/my-repo/:_authToken=token
//us-central1-npm.pkg.dev/my-other-project/my-other-repo/:_authToken=token
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub out the logic for fetching Application Default Credentials
			origFindDefaultCredentials := findDefaultCredentials
			findDefaultCredentials = func() (string, error) {
				return "token", tc.tokenError
			}
			defer func() {
				findDefaultCredentials = origFindDefaultCredentials
			}()

			t.Setenv("GOOGLE_EXPERIMENTAL_AR_AUTH_ENABLED", "True")

			// set up the application root dir
			tempRoot := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))
			if tc.projectNpmrc != "" {
				filepath := filepath.Join(tempRoot, ".npmrc")
				os.WriteFile(filepath, []byte(tc.projectNpmrc), 0664)
			}

			// set up the $HOME dir
			t.Setenv("HOME", t.TempDir())
			filepath := filepath.Join(ctx.HomeDir(), ".npmrc")
			if tc.fileExists {
				os.WriteFile(filepath, []byte{}, 0664)
			}

			if err := GenerateNPMConfig(ctx); err != nil {
				t.Fatalf("generating config: %v", err)
			}

			config, err := os.ReadFile(filepath)
			if err != nil && tc.wantConfig != "" {
				t.Fatalf("reading file %s: %v", filepath, err)
			}

			if diff := cmp.Diff(tc.wantConfig, string(config)); diff != "" {
				t.Errorf("TestGenerateNPMConfigMetrics unexpected config (+got, -want):\n %v", diff)
			} else {
				successfulCredGens++
			}

			if buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Value() != successfulCredGens {
				t.Errorf("TestGenerateNPMConfigMetrics incorrect cred gen count: got %v, want %v",
					buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Value(), successfulCredGens)
			}
		})
	}
}

func TestNpmRegistryRegexp(t *testing.T) {
	testCases := []struct {
		name  string
		npmrc string
		want  []string
	}{
		{
			name: "empty string",
		},
		{
			name:  "npm.org repo",
			npmrc: "//registry.npmjs.org/:_authToken=${NPM_TOKEN}",
		},
		{
			name:  "unscoped AR repo",
			npmrc: "registry=https://us-west1-npm.pkg.dev/my-project/my-repo/",
			want: []string{
				"registry=https://us-west1-npm.pkg.dev/my-project/my-repo/",
				"",
				"//us-west1-npm.pkg.dev/my-project/my-repo/",
			},
		},
		{
			name:  "scoped AR repo",
			npmrc: "@myscope:registry=https://us-central1-npm.pkg.dev/my-other-project/my-other-repo/",
			want: []string{
				"@myscope:registry=https://us-central1-npm.pkg.dev/my-other-project/my-other-repo/",
				"@myscope:",
				"//us-central1-npm.pkg.dev/my-other-project/my-other-repo/",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := npmRegistryRegexp.FindStringSubmatch(tc.npmrc)
			if diff := cmp.Diff(tc.want, matches); diff != "" {
				t.Errorf("unexpected config (+got, -want):\n %v", diff)
			}
		})
	}
}

func TestGenerateYarnConfig(t *testing.T) {
	t.Cleanup(buildermetrics.Reset)
	successfulCredGens := int64(0)
	testCases := []struct {
		name       string
		fileExists bool
		tokenError error
		projYarnrc string
		wantConfig string
	}{
		{
			name:       "user .yarnrc.yml already exists",
			fileExists: true,
		},
		{
			name:       "credential error",
			tokenError: fmt.Errorf("Error fetching token"),
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: https://registry.npmjs.org/
    npmAlwaysAuth: true
`,
		},
		{
			name: "project .yarnrc.yml with npmjs.org config",
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: https://registry.npmjs.org/
    npmAlwaysAuth: true
`,
		},
		{
			name: "project .yarnrc.yml with AR config",
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: https://us-central1-npm.pkg.dev/project/repo/
    npmAlwaysAuth: true
`,
			wantConfig: `npmRegistries:
  https://us-central1-npm.pkg.dev/project/repo/:
    npmAlwaysAuth: true
    npmAuthToken: token
`,
		},
		{
			name: "project .yarnrc.yml with multiple AR configs",
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: https://us-central1-npm.pkg.dev/project/repo/
    npmAlwaysAuth: true
  scope2:
    npmRegistryServer: https://us-west1-npm.pkg.dev/project/repo/
    npmAlwaysAuth: true
`,
			wantConfig: `npmRegistries:
  https://us-central1-npm.pkg.dev/project/repo/:
    npmAlwaysAuth: true
    npmAuthToken: token
  https://us-west1-npm.pkg.dev/project/repo/:
    npmAlwaysAuth: true
    npmAuthToken: token
`,
		},
		{
			name: "project .yarnrc.yml with AR and non-AR configs",
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: https://us-central1-npm.pkg.dev/project/repo/
    npmAlwaysAuth: true
  scope2:
    npmRegistryServer: https://registry.npmjs.org/
    npmAlwaysAuth: true
`,
			wantConfig: `npmRegistries:
  https://us-central1-npm.pkg.dev/project/repo/:
    npmAlwaysAuth: true
    npmAuthToken: token
`,
		},
		{
			name: "project with .yarnrc.yml with empty AR config",
			projYarnrc: `npmScopes:
  scope1:
    npmRegistryServer: 
    npmAlwaysAuth: true
`,
		},
		{
			name: "project without .yarnrc.yml",
		},
		{
			name: "project with empty .yarnrc.yml",
			projYarnrc: `
`,
		},
		{
			name: "project with .yarnrc.yml without npm scopes", // test case adds a different yaml property to project config
			projYarnrc: `enableColors: true
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// stub out the logic for fetching Application Default Credentials
			origFindDefaultCredentials := findDefaultCredentials
			findDefaultCredentials = func() (string, error) {
				return "token", tc.tokenError
			}
			defer func() {
				findDefaultCredentials = origFindDefaultCredentials
			}()

			// set up the application root dir
			tempRoot := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(tempRoot))
			file := filepath.Join(tempRoot, ".yarnrc.yml")

			if tc.projYarnrc != "" {
				os.WriteFile(file, []byte(tc.projYarnrc), 0664)
			}

			// set up the $HOME dir
			t.Setenv("HOME", t.TempDir())
			userConfigFile := filepath.Join(ctx.HomeDir(), ".yarnrc.yml")
			if tc.fileExists {
				os.WriteFile(userConfigFile, []byte{}, 0664)
			}

			if err := GenerateYarnConfig(ctx); err != nil {
				t.Fatalf("Error generating config: %v", err)
			}

			config, err := os.ReadFile(userConfigFile)
			if err != nil && tc.wantConfig != "" {
				t.Errorf("Reading file %s: %v", userConfigFile, err)
			}

			if diff := cmp.Diff(tc.wantConfig, string(config)); diff != "" {
				t.Errorf("unexpected config (+got, -want):\n %v", diff)
			} else if tc.wantConfig != "" {
				successfulCredGens++
			}

			if buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Value() != successfulCredGens {
				t.Errorf("TestGenerateYarnConfig incorrect cred gen count: got %v, want %v",
					buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Value(), successfulCredGens)
			}
		})
	}
}
