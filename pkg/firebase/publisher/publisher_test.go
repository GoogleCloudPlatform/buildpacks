// Copyright 2023 Google LLC
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

package publisher

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"google3/third_party/golang/cmp/cmpopts/cmpopts"
	"google3/third_party/golang/protobuf/v2/proto/proto"
	"gopkg.in/yaml.v2"

	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
)

var (
	appHostingYAMLPath string = testdata.MustGetPath("testdata/apphosting.yaml")
	bundleYAMLPath     string = testdata.MustGetPath("testdata/bundle.yaml")
	latestSecretName   string = "projects/test-project/secrets/secretID/versions/12"
)

func TestPublish(t *testing.T) {
	testDir := t.TempDir()

	testCases := []struct {
		desc                   string
		appHostingYAMLFilePath string
		wantBuildSchema        buildSchema
	}{{
		desc:                   "Publish apphosting.yaml, bundle.yaml",
		appHostingYAMLFilePath: appHostingYAMLPath,
		wantBuildSchema: buildSchema{
			RunConfig: &apphostingschema.RunConfig{
				CPU:          proto.Float32(3),
				MemoryMiB:    proto.Int32(1024),
				Concurrency:  proto.Int32(100),
				MaxInstances: proto.Int32(4),
				MinInstances: proto.Int32(0),
			},
			Env: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{
					Variable:     "API_URL",
					Value:        "api.service.com",
					Availability: []string{"BUILD", "RUNTIME"},
				},
				apphostingschema.EnvironmentVariable{
					Variable:     "STORAGE_BUCKET",
					Value:        "mybucket.appspot.com",
					Availability: []string{"RUNTIME"},
				},
				apphostingschema.EnvironmentVariable{
					Variable:     "API_KEY",
					Secret:       "projects/test-project/secrets/secretID/versions/12",
					Availability: []string{"BUILD"},
				},
			},
		}},
		{
			desc:                   "Handle nonexistent apphosting.yaml",
			appHostingYAMLFilePath: "nonexistent",
			wantBuildSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					Concurrency:  proto.Int32(80),
					MaxInstances: proto.Int32(100),
					MinInstances: proto.Int32(0),
				},
			}},
	}

	// Testing happy paths
	for i, test := range testCases {
		outputFilePath := fmt.Sprintf("%s/outputhappy%d", testDir, i)

		err := Publish(test.appHostingYAMLFilePath, bundleYAMLPath, outputFilePath)
		if err != nil {
			t.Errorf("Error in test '%v'. Error was %v", test.desc, err)
		}

		actualBuildSchemaData, err := ioutil.ReadFile(outputFilePath)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		var actualBuildSchema buildSchema
		err = yaml.Unmarshal(actualBuildSchemaData, &actualBuildSchema)

		if err != nil {
			t.Errorf("error unmarshalling %q as YAML: %v", actualBuildSchemaData, err)
		}

		sort := cmpopts.SortSlices(func(a, b apphostingschema.EnvironmentVariable) bool { return a.Variable < b.Variable })
		if diff := cmp.Diff(test.wantBuildSchema, actualBuildSchema, sort); diff != "" {
			t.Errorf("Unexpected YAML for test %v (+got, -want):\n%v", test.desc, diff)
		}
	}
}

func TestToBuildSchemaRunConfig(t *testing.T) {
	tests := []struct {
		desc                  string
		inputAppHostingSchema apphostingschema.AppHostingSchema
		expectedSchema        buildSchema
	}{
		{
			desc:                  "Empty AppHostingSchema",
			inputAppHostingSchema: apphostingschema.AppHostingSchema{},
			expectedSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(float32(defaultCPU)),
					MemoryMiB:    &defaultMemory,
					Concurrency:  &defaultConcurrency,
					MaxInstances: &defaultMaxInstances,
					MinInstances: proto.Int32(0),
				},
			},
		},
		{
			desc: "Full AppHostingSchema",
			inputAppHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    proto.Int32(2048),
					Concurrency:  proto.Int32(2),
					MaxInstances: proto.Int32(5),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
				},
			},
			expectedSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    proto.Int32(2048),
					Concurrency:  proto.Int32(2),
					MaxInstances: proto.Int32(5),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc: "Partial AppHostingSchema",
			inputAppHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:         proto.Float32(1000),
					Concurrency: proto.Int32(2),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
				},
			},
			expectedSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    &defaultMemory,
					Concurrency:  proto.Int32(2),
					MaxInstances: &defaultMaxInstances,
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			bundleSchema := outputBundleSchema{}
			result := toBuildSchema(tc.inputAppHostingSchema, bundleSchema)
			if diff := cmp.Diff(tc.expectedSchema, result); diff != "" {
				t.Errorf("toBuildSchema(%s) (-want +got):\n%s", tc.desc, diff)
			}
		})
	}
}
