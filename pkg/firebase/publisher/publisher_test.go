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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

var (
	appHostingCompleteYAMLPath string = testdata.MustGetPath("testdata/apphosting_complete.yaml")
	appHostingInvalidYAMLPath  string = testdata.MustGetPath("testdata/apphosting_invalid.yaml")
	appHostingCompleteEnvPath  string = testdata.MustGetPath("testdata/apphosting_complete.env")
	appHostingReservedEnvPath  string = testdata.MustGetPath("testdata/apphosting_reserved.env")
	bundleYAMLPath             string = testdata.MustGetPath("testdata/bundle.yaml")
)

func int32Ptr(i int) *int32 {
	v := new(int32)
	*v = int32(i)
	return v
}

func toString(buildSchema buildSchema) string {
	data, _ := json.MarshalIndent(buildSchema, "", "  ")
	return string(data)
}

func TestPublish(t *testing.T) {
	testDir := t.TempDir()
	outputFilePath := testDir + "/output"

	testCasesHappy := []struct {
		desc                   string
		appHostingYAMLFilePath string
		appHostingEnvFilePath  string
		wantBuildSchema        buildSchema
	}{{
		desc:                   "Publish apphosting.yaml, bundle.yaml, and apphosting.env",
		appHostingYAMLFilePath: appHostingCompleteYAMLPath,
		appHostingEnvFilePath:  appHostingCompleteEnvPath,
		wantBuildSchema: buildSchema{
			BackendResources: backendResources{
				CPU:          int32Ptr(3),
				Memory:       int32Ptr(512),
				Concurrency:  int32Ptr(100),
				MaxInstances: int32Ptr(4),
			},
			Runtime: runtime{
				EnvVariables: map[string]string{
					"API_URL":     "api.service.com",
					"ENVIRONMENT": "staging",
				},
			},
		}},
		{
			desc:                   "Handle nonexistent apphosting.yaml",
			appHostingYAMLFilePath: "nonexistent",
			appHostingEnvFilePath:  appHostingCompleteEnvPath,
			wantBuildSchema: buildSchema{
				BackendResources: backendResources{
					CPU:          int32Ptr(1),
					Memory:       int32Ptr(512),
					Concurrency:  int32Ptr(80),
					MaxInstances: int32Ptr(100),
				},
				Runtime: runtime{
					EnvVariables: map[string]string{
						"API_URL":     "api.service.com",
						"ENVIRONMENT": "staging",
					},
				},
			}},
	}

	testCasesError := []struct {
		desc                   string
		appHostingYAMLFilePath string
		appHostingEnvFilePath  string
		wantError              string
	}{{
		desc:                   "Throw an error parsing apphosting.yaml when values are invalid",
		appHostingYAMLFilePath: appHostingInvalidYAMLPath,
		appHostingEnvFilePath:  appHostingCompleteEnvPath,
		wantError:              "concurrency",
	}}

	// Testing happy paths
	for _, test := range testCasesHappy {
		err := Publish(test.appHostingYAMLFilePath, bundleYAMLPath, test.appHostingEnvFilePath, outputFilePath)
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
			t.Errorf("Unexpected YAML for test %v (+got, -want): \n %v", test.desc, diff)
		}
	}

	// Testing error paths
	for _, test := range testCasesError {
		err := Publish(test.appHostingYAMLFilePath, bundleYAMLPath, test.appHostingEnvFilePath, outputFilePath)
		if err == nil {
			t.Errorf("calling Publish did not produce an error for test %q", test.desc)
		}

		if !strings.Contains(err.Error(), test.wantError) {
			t.Errorf("error not in expected format for test %q.\nGot: %v\nWant: %v", test.desc, err, test.wantError)
		}
	}
}

func TestValidateAppHostingYAMLFields(t *testing.T) {
	testCasesError := []struct {
		desc             string
		appHostingSchema appHostingSchema
		wantError        bool
	}{{
		desc: "Throw no error when schema is valid",
		appHostingSchema: appHostingSchema{
			BackendResources: backendResources{
				CPU:          int32Ptr(7),
				Memory:       int32Ptr(1024),
				Concurrency:  int32Ptr(500),
				MaxInstances: int32Ptr(4),
			},
		},
		wantError: false,
	},
		{
			desc: "Throw an error when CPU value is invalid",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					CPU: int32Ptr(50),
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when Memory value is invalid",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					Memory: int32Ptr(40000),
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when concurrency value is invalid",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					Concurrency: int32Ptr(2000),
				},
			},
			wantError: true,
		},
		{
			desc: "Throw an error when maxInstances value is invalid",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					MaxInstances: int32Ptr(101),
				},
			},
			wantError: true,
		}}

	for _, test := range testCasesError {
		err := validateAppHostingYAMLFields(test.appHostingSchema)
		if err != nil != test.wantError {
			t.Errorf("validateAppHostingYAMLFields(%q) = %v, want %v", test.desc, err, test.wantError)
		}
	}
}

func TestToBuildSchemaBackendResources(t *testing.T) {
	tests := []struct {
		name             string
		appHostingSchema appHostingSchema
		expected         buildSchema
	}{
		{
			name:             "Empty AppHostingSchema",
			appHostingSchema: appHostingSchema{},
			expected: buildSchema{
				BackendResources: backendResources{
					CPU:          &defaultCPU,
					Memory:       &defaultMemory,
					Concurrency:  &defaultConcurrency,
					MaxInstances: &defaultMaxInstances,
				},
			},
		},
		{
			name: "Full AppHostingSchema",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					CPU:          int32Ptr(1000),
					Memory:       int32Ptr(2048),
					Concurrency:  int32Ptr(2),
					MaxInstances: int32Ptr(5),
				},
			},
			expected: buildSchema{
				BackendResources: backendResources{
					CPU:          int32Ptr(1000),
					Memory:       int32Ptr(2048),
					Concurrency:  int32Ptr(2),
					MaxInstances: int32Ptr(5),
				},
			},
		},
		{
			name: "Partial AppHostingSchema",
			appHostingSchema: appHostingSchema{
				BackendResources: backendResources{
					CPU:         int32Ptr(1000),
					Concurrency: int32Ptr(2),
				},
			},
			expected: buildSchema{
				BackendResources: backendResources{
					CPU:          int32Ptr(1000),
					Memory:       &defaultMemory,
					Concurrency:  int32Ptr(2),
					MaxInstances: &defaultMaxInstances,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundleSchema := outputBundleSchema{}
			result := toBuildSchema(tc.appHostingSchema, bundleSchema, map[string]string{})
			if !cmp.Equal(result, tc.expected) {
				t.Errorf("toBuildSchema(%q) = %+v, want = %+v", tc.name, result, tc.expected)
			}
		})
	}
}
