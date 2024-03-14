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
	"encoding/json"
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
	appHostingCompleteYAMLPath string = testdata.MustGetPath("testdata/apphosting_complete.yaml")
	envPath                    string = testdata.MustGetPath("testdata/env")
	bundleYAMLPath             string = testdata.MustGetPath("testdata/bundle.yaml")
)

func toString(buildSchema buildSchema) string {
	data, _ := json.MarshalIndent(buildSchema, "", "  ")
	return string(data)
}

func TestPublish(t *testing.T) {
	testDir := t.TempDir()

	testCasesHappy := []struct {
		desc                   string
		appHostingYAMLFilePath string
		envFilePath            string
		wantBuildSchema        buildSchema
	}{{
		desc:                   "Publish apphosting.yaml, bundle.yaml, and apphosting.env",
		appHostingYAMLFilePath: appHostingCompleteYAMLPath,
		envFilePath:            envPath,
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
					Availability: []string{"RUNTIME"},
				},
				apphostingschema.EnvironmentVariable{
					Variable:     "ENVIRONMENT",
					Value:        "staging",
					Availability: []string{"RUNTIME"},
				},
				apphostingschema.EnvironmentVariable{
					Variable:     "MULTILINE_ENV_VAR",
					Value:        "line 1\nline 2",
					Availability: []string{"RUNTIME"},
				},
			},
			Runtime: &runtime{
				EnvVariables: map[string]string{
					"API_URL":           "api.service.com",
					"ENVIRONMENT":       "staging",
					"MULTILINE_ENV_VAR": "line 1\nline 2",
				},
			},
		}},
		{
			desc:                   "Handle nonexistent apphosting.yaml",
			appHostingYAMLFilePath: "nonexistent",
			envFilePath:            envPath,
			wantBuildSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					Concurrency:  proto.Int32(80),
					MaxInstances: proto.Int32(100),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{
						Variable:     "ENVIRONMENT",
						Value:        "staging",
						Availability: []string{"RUNTIME"},
					},
					apphostingschema.EnvironmentVariable{
						Variable:     "MULTILINE_ENV_VAR",
						Value:        "line 1\nline 2",
						Availability: []string{"RUNTIME"},
					},
					apphostingschema.EnvironmentVariable{
						Variable:     "API_URL",
						Value:        "api.service.com",
						Availability: []string{"RUNTIME"},
					},
				},
				Runtime: &runtime{
					EnvVariables: map[string]string{
						"API_URL":           "api.service.com",
						"ENVIRONMENT":       "staging",
						"MULTILINE_ENV_VAR": "line 1\nline 2",
					},
				},
			}},
		{
			desc:                   "Handle nonexistent apphosting.env",
			appHostingYAMLFilePath: appHostingCompleteYAMLPath,
			envFilePath:            "nonexistent",
			wantBuildSchema: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
			}},
	}

	// Testing happy paths
	for i, test := range testCasesHappy {
		outputFilePath := fmt.Sprintf("%s/outputhappy%d", testDir, i)

		err := Publish(test.appHostingYAMLFilePath, bundleYAMLPath, test.envFilePath, outputFilePath)
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
		name             string
		appHostingSchema apphostingschema.AppHostingSchema
		expected         buildSchema
	}{
		{
			name:             "Empty AppHostingSchema",
			appHostingSchema: apphostingschema.AppHostingSchema{},
			expected: buildSchema{
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
			name: "Full AppHostingSchema",
			appHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    proto.Int32(2048),
					Concurrency:  proto.Int32(2),
					MaxInstances: proto.Int32(5),
					MinInstances: proto.Int32(0),
				},
			},
			expected: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    proto.Int32(2048),
					Concurrency:  proto.Int32(2),
					MaxInstances: proto.Int32(5),
					MinInstances: proto.Int32(0),
				},
			},
		},
		{
			name: "Partial AppHostingSchema",
			appHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:         proto.Float32(1000),
					Concurrency: proto.Int32(2),
				},
			},
			expected: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          proto.Float32(1000),
					MemoryMiB:    &defaultMemory,
					Concurrency:  proto.Int32(2),
					MaxInstances: &defaultMaxInstances,
					MinInstances: proto.Int32(0),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundleSchema := outputBundleSchema{}
			result := toBuildSchema(tc.appHostingSchema, bundleSchema, map[string]string{})
			if diff := cmp.Diff(tc.expected, result); diff != "" {
				t.Errorf("toBuildSchema(%s) (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}
