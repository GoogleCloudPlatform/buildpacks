package apphostingschema

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"google3/third_party/golang/protobuf/v2/proto/proto"
)

func TestReadAndValidateAppHostingSchemaFromFile(t *testing.T) {
	testCases := []struct {
		desc                 string
		inputAppHostingYAML  string
		wantAppHostingSchema AppHostingSchema
		wantErr              bool
	}{
		{
			desc:                "Read properly formatted app hosting YAML schema properly",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_valid.yaml"),
			wantAppHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "myApiKeySecret", Availability: []string{"BUILD"}},
					EnvironmentVariable{Variable: "PINNED_API_KEY", Secret: "myApiKeySecret@5"}},
			},
		},
		{
			desc:                "Read YAML schema missing an env section properly",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_missingenv.yaml"),
			wantAppHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
				},
			},
		},
		{
			desc:                 "Return an empty schema when the file doesn't exist",
			inputAppHostingYAML:  testdata.MustGetPath("testdata/nonexistant.yaml"), // File doesn't exist
			wantAppHostingSchema: AppHostingSchema{},
		},
		{
			desc:                "Throw an error when an env field contains both a value and a secret",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidenv_valuesecret.yaml"),
			wantErr:             true,
		},
		{
			desc:                "Throw an error when an env field contains an invalid availability value",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidenv_availability.yaml"),
			wantErr:             true,
		},
		{
			desc:                "Throw an error when a run config field contains an invalid value",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidrunconfig.yaml"),
			wantErr:             true,
		},
	}

	for _, test := range testCases {
		s, err := ReadAndValidateAppHostingSchemaFromFile(test.inputAppHostingYAML)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("unexpected error for ReadAppHostingSchemaFromFile(%q): %v", test.desc, err)
			}

			if diff := cmp.Diff(test.wantAppHostingSchema, s); diff != "" {
				t.Errorf("unexpected YAML for test %q, (-want, +got):\n%v", test.desc, diff)
			}

			// Error Path
		} else {
			if err == nil {
				t.Errorf("ReadAppHostingSchemaFromFile(%q) = %v, want error", test.desc, err)
			}
		}
	}
}

func TestSanitize(t *testing.T) {
	testCases := []struct {
		desc        string
		inputSchema AppHostingSchema
		wantSchema  AppHostingSchema
	}{
		{
			desc: "Sanitize keys properly",
			inputSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "K_SERVICE", Secret: "secretID"},
					EnvironmentVariable{Variable: "FIREBASE_ID", Value: "firebaseId", Availability: []string{"BUILD"}},
					EnvironmentVariable{Variable: "X_FIREBASE_RESERVED", Value: "value"},
					EnvironmentVariable{Variable: "MISSING_AVAILABILITY", Value: "projects/test-project/secrets/secretID"},
				},
			},
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "FIREBASE_ID", Value: "firebaseId", Availability: []string{"BUILD"}},
					EnvironmentVariable{Variable: "MISSING_AVAILABILITY", Value: "projects/test-project/secrets/secretID", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc: "Remove no keys when all env vars are valid",
			inputSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "ENVIRONMENT", Secret: "staging", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "ENVIRONMENT", Secret: "staging", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc: "Properly sanitize when environment variables are missing",
			inputSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
			},
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					MemoryMiB: proto.Int32(1024),
				},
			},
		},
	}

	for _, test := range testCases {
		Sanitize(&test.inputSchema)
		if diff := cmp.Diff(test.wantSchema, test.inputSchema); diff != "" {
			t.Errorf("unexpected sanitized envVars for test %q (-want, +got):\n%v", test.desc, diff)
		}
	}
}

func TestMergeWithEnvironmentSpecificYAML(t *testing.T) {
	testCases := []struct {
		desc               string
		appHostingSchema   AppHostingSchema
		appHostingYAMLPath string
		environmentName    string
		wantSchema         AppHostingSchema
	}{
		{
			desc: "Merge apphosting.yaml and apphosting.<environmentName>.yaml properly",
			appHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
			appHostingYAMLPath: testdata.MustGetPath("testdata/apphosting.yaml"),
			environmentName:    "staging",
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(5),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
					EnvironmentVariable{Variable: "DATABASE_URL", Secret: "secretStagingDatabaseURL"},
				},
			},
		},
		{
			desc: "Don't modify apphosting.yaml when apphosting.<environmentName>.yaml is empty",
			appHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
			appHostingYAMLPath: testdata.MustGetPath("testdata/apphosting.yaml"),
			environmentName:    "empty",
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
		},
		{
			desc: "Don't modify apphosting.yaml when environment name isn't passed in",
			appHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
			appHostingYAMLPath: testdata.MustGetPath("testdata/apphosting.yaml"),
			environmentName:    "",
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
		},
		{
			desc: "Use apphosting.yaml when apphosting.<environmentName>.yaml is not found",
			appHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
			appHostingYAMLPath: testdata.MustGetPath("testdata/apphosting.yaml"),
			environmentName:    "missingfile",
			wantSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
					Concurrency:  proto.Int32(100),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "secretIDforAPI"},
				},
			},
		},
	}

	for _, test := range testCases {
		if err := MergeWithEnvironmentSpecificYAML(&test.appHostingSchema, test.appHostingYAMLPath, test.environmentName); err != nil {
			t.Fatalf("unexpected error for TestMergeWithEnvironmentSpecificYAML(%q): %v", test.desc, err)
		}

		if diff := cmp.Diff(test.wantSchema, test.appHostingSchema); diff != "" {
			t.Errorf("unexpected merged apphosting schema for test %q (-want, +got):\n%v", test.desc, diff)
		}
	}

}

func TestWriteToFile(t *testing.T) {
	testDir := t.TempDir()

	testCases := []struct {
		desc        string
		inputSchema AppHostingSchema
	}{
		{
			desc: "Write properly formatted app hosting YAML schema correctly",
			inputSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:       proto.Float32(3),
					MemoryMiB: proto.Int32(1024),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "MISSING_AVAILABILITY", Value: "projects/test-project/secrets/secretID", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc: "Write schema missing RunConfig correctly",
			inputSchema: AppHostingSchema{
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					EnvironmentVariable{Variable: "MISSING_AVAILABILITY", Value: "projects/test-project/secrets/secretID", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc: "Write schema missing Env correctly",
			inputSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:       proto.Float32(3),
					MemoryMiB: proto.Int32(1024),
				},
			},
		},
		{
			desc:        "Write empty schema correctly",
			inputSchema: AppHostingSchema{},
		},
	}

	for i, test := range testCases {
		outputFilePath := fmt.Sprintf("%s/output%d", testDir, i)

		err := test.inputSchema.WriteToFile(outputFilePath)
		if err != nil {
			t.Errorf("error in test '%v'. Error was %v", test.desc, err)
		}

		actualSchema, err := ReadAndValidateAppHostingSchemaFromFile(outputFilePath)
		if err != nil {
			t.Errorf("error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.inputSchema, actualSchema); diff != "" {
			t.Errorf("unexpected schema for test %q, (-want, +got):\n%v", test.desc, diff)
		}
	}
}
