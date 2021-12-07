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
	"io/ioutil"
	"path/filepath"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"google3/third_party/golang/cmp/cmp"
)

func TestGeneratePythonConfig(t *testing.T) {
	testCases := []struct {
		name          string
		arAuthEnabled string
		fileExists    bool
		tokenError    error
		wantConfig    string
	}{
		{
			name:          "Feature Disabled",
			arAuthEnabled: "False",
			fileExists:    false,
			wantConfig:    "",
		},
		{
			name:          ".netrc already exists",
			arAuthEnabled: "True",
			fileExists:    true,
			wantConfig:    "",
		},
		{
			name:          "credential error",
			arAuthEnabled: "True",
			fileExists:    false,
			tokenError:    fmt.Errorf("Error fetching token"),
			wantConfig:    "",
		},
		{
			name:          ".netrc created",
			arAuthEnabled: "True",
			fileExists:    false,
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
machine europe-west1-python.pkg.dev login oauth2accesstoken password token
machine europe-west2-python.pkg.dev login oauth2accesstoken password token
machine europe-west3-python.pkg.dev login oauth2accesstoken password token
machine europe-west4-python.pkg.dev login oauth2accesstoken password token
machine europe-west5-python.pkg.dev login oauth2accesstoken password token
machine europe-west6-python.pkg.dev login oauth2accesstoken password token
machine northamerica-northeast1-python.pkg.dev login oauth2accesstoken password token
machine northamerica-northeast2-python.pkg.dev login oauth2accesstoken password token
machine southamerica-east1-python.pkg.dev login oauth2accesstoken password token
machine us-central1-python.pkg.dev login oauth2accesstoken password token
machine us-python.pkg.dev login oauth2accesstoken password token
machine us-east1-python.pkg.dev login oauth2accesstoken password token
machine us-east4-python.pkg.dev login oauth2accesstoken password token
machine us-west1-python.pkg.dev login oauth2accesstoken password token
machine us-west2-python.pkg.dev login oauth2accesstoken password token
machine us-west3-python.pkg.dev login oauth2accesstoken password token
machine us-west4-python.pkg.dev login oauth2accesstoken password token
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

			t.Setenv("GOOGLE_EXPERIMENTAL_AR_AUTH_ENABLED", tc.arAuthEnabled)

			ctx := gcp.NewContext()

			filepath := filepath.Join(tempHome, ".netrc")
			if tc.fileExists {
				f := ctx.CreateFile(filepath)
				f.Close()
			}

			if err := GeneratePythonConfig(ctx); err != nil {
				t.Fatalf("Generating config: %v", err)
			}

			config, err := ioutil.ReadFile(filepath)
			if err != nil && tc.wantConfig != "" {
				t.Fatalf("Reading file %s: %v", filepath, err)
			}

			if diff := cmp.Diff(tc.wantConfig, string(config)); diff != "" {
				t.Errorf("unexpected config (+got, -want):\n %v", diff)
			}
		})
	}
}
