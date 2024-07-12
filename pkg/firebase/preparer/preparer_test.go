package preparer

import (
	"context"
	"hash/crc32"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/fakesecretmanager"
	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/envvars"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google3/third_party/golang/protobuf/v2/proto/proto"
)

var (
	appHostingYAMLPath   string = testdata.MustGetPath("testdata/apphosting.yaml")
	latestSecretName     string = "projects/test-project/secrets/secretID/versions/12"
	pinnedSecretName     string = "projects/test-project/secrets/secretID/versions/11"
	secretString         string = "secretString"
	secretStringChecksum int64  = int64(crc32.Checksum([]byte(secretString), crc32.MakeTable(crc32.Castagnoli)))
)

func TestPrepare(t *testing.T) {
	testDir := t.TempDir()
	outputFilePathYAML := testDir + "/outputYAML"
	outputFilePathEnv := testDir + "/outputEnv"
	outputFilePathBuildpackConfig := testDir + "/outputBuildpackConfig"

	testCases := []struct {
		desc               string
		appHostingYAMLPath string
		projectID          string
		wantEnvMap         map[string]string
		wantSchema         apphostingschema.AppHostingSchema
	}{
		{
			desc:               "properly prepare apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			wantEnvMap: map[string]string{
				"API_URL":                "api.service.com",
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
				"VAR_NUMBER":             "12345",
				"API_KEY":                secretString,
				"PINNED_API_KEY":         secretString,
				"VERBOSE_API_KEY":        secretString,
				"PINNED_VERBOSE_API_KEY": secretString,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD"}},
					apphostingschema.EnvironmentVariable{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
					apphostingschema.EnvironmentVariable{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc:               "non-existent apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			wantEnvMap:         map[string]string{},
			wantSchema:         apphostingschema.AppHostingSchema{},
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		SecretVersionResponses: map[string]fakesecretmanager.GetSecretVersionResponse{
			"projects/test-project/secrets/secretID/versions/latest": fakesecretmanager.GetSecretVersionResponse{
				SecretVersion: &smpb.SecretVersion{
					Name:  latestSecretName,
					State: smpb.SecretVersion_ENABLED,
				},
			},
		},
		AccessSecretVersionResponses: map[string]fakesecretmanager.AccessSecretVersionResponse{
			pinnedSecretName: fakesecretmanager.AccessSecretVersionResponse{
				Response: &smpb.AccessSecretVersionResponse{
					Payload: &smpb.SecretPayload{
						Data:       []byte(secretString),
						DataCrc32C: &secretStringChecksum,
					},
				},
			},
			latestSecretName: fakesecretmanager.AccessSecretVersionResponse{
				Response: &smpb.AccessSecretVersionResponse{
					Payload: &smpb.SecretPayload{
						Data:       []byte(secretString),
						DataCrc32C: &secretStringChecksum,
					},
				},
			},
		},
	}

	// Testing happy paths
	for _, test := range testCases {
		opts := Options{
			SecretClient:                  fakeSecretClient,
			AppHostingYAMLPath:            test.appHostingYAMLPath,
			ProjectID:                     test.projectID,
			AppHostingYAMLOutputFilePath:  outputFilePathYAML,
			EnvDereferencedOutputFilePath: outputFilePathEnv,
			BackendRootDirectory:          "",
			BuildpackConfigOutputFilePath: outputFilePathBuildpackConfig,
		}

		if err := Prepare(context.Background(), opts); err != nil {
			t.Errorf("Error in test '%v'. Error was %v", test.desc, err)
		}

		// Check dereferenced secret material env file
		actualEnvMapDereferenced, err := envvars.Read(outputFilePathEnv)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMap, actualEnvMapDereferenced); diff != "" {
			t.Errorf("Unexpected env map for test %v (+got, -want):\n%v", test.desc, diff)
		}

		// Check app hosting schema
		actualAppHostingSchema, err := apphostingschema.ReadAndValidateAppHostingSchemaFromFile(outputFilePathYAML)
		if err != nil {
			t.Errorf("reading in and validating apphosting.yaml at path %v: %v", outputFilePathYAML, err)
		}

		if diff := cmp.Diff(test.wantSchema, actualAppHostingSchema); diff != "" {
			t.Errorf("unexpected prepared YAML schema for test %q (+got, -want):\n%v", test.desc, diff)
		}
	}
}
