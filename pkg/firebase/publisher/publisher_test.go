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
	"gopkg.in/yaml.v2"

	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
)

var (
	appHostingCompleteYAMLPath string = testdata.MustGetPath("testdata/apphosting_complete.yaml")
	envPath                    string = testdata.MustGetPath("testdata/env")
	bundleYAMLPath             string = testdata.MustGetPath("testdata/bundle.yaml")
)

func int32Ptr(i int) *int32 {
	v := new(int32)
	*v = int32(i)
	return v
}

func float32Ptr(i int32) *float32 {
	v := new(float32)
	*v = float32(i)
	return v
}

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
				CPU:          float32Ptr(3),
				MemoryMiB:    int32Ptr(1024),
				Concurrency:  int32Ptr(100),
				MaxInstances: int32Ptr(4),
				MinInstances: int32Ptr(0),
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
					CPU:          float32Ptr(1),
					MemoryMiB:    int32Ptr(512),
					Concurrency:  int32Ptr(80),
					MaxInstances: int32Ptr(100),
					MinInstances: int32Ptr(0),
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
					CPU:          float32Ptr(3),
					MemoryMiB:    int32Ptr(1024),
					Concurrency:  int32Ptr(100),
					MaxInstances: int32Ptr(4),
					MinInstances: int32Ptr(0),
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

		if diff := cmp.Diff(test.wantBuildSchema, actualBuildSchema); diff != "" {
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
					CPU:          float32Ptr(defaultCPU),
					MemoryMiB:    &defaultMemory,
					Concurrency:  &defaultConcurrency,
					MaxInstances: &defaultMaxInstances,
					MinInstances: int32Ptr(0),
				},
			},
		},
		{
			name: "Full AppHostingSchema",
			appHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          float32Ptr(1000),
					MemoryMiB:    int32Ptr(2048),
					Concurrency:  int32Ptr(2),
					MaxInstances: int32Ptr(5),
					MinInstances: int32Ptr(0),
				},
			},
			expected: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          float32Ptr(1000),
					MemoryMiB:    int32Ptr(2048),
					Concurrency:  int32Ptr(2),
					MaxInstances: int32Ptr(5),
					MinInstances: int32Ptr(0),
				},
			},
		},
		{
			name: "Partial AppHostingSchema",
			appHostingSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:         float32Ptr(1000),
					Concurrency: int32Ptr(2),
				},
			},
			expected: buildSchema{
				RunConfig: &apphostingschema.RunConfig{
					CPU:          float32Ptr(1000),
					MemoryMiB:    &defaultMemory,
					Concurrency:  int32Ptr(2),
					MaxInstances: &defaultMaxInstances,
					MinInstances: int32Ptr(0),
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
