package preparer

import (
	"context"
	"encoding/json"
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
		environmentName    string
		serverSideEnvVars  []apphostingschema.EnvironmentVariable
		wantEnvMap         map[string]string
		wantSchema         apphostingschema.AppHostingSchema
	}{
		{
			desc:               "properly prepare apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			environmentName:    "staging",
			wantEnvMap: map[string]string{
				"API_URL":                 "api.staging.service.com",
				"VAR_QUOTED_SPECIAL":      "api2.service.com::",
				"VAR_SPACED":              "api3 - service -  com",
				"VAR_SINGLE_QUOTES":       "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":       "\"api4.service.com\"",
				"MULTILINE_VAR":           "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
				"VAR_NUMBER":              "12345",
				"API_KEY":                 secretString,
				"PINNED_API_KEY":          secretString,
				"VERBOSE_API_KEY":         secretString,
				"PINNED_VERBOSE_API_KEY":  secretString,
				"STAGING_SECRET_VARIABLE": secretString,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(2),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}},
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
					apphostingschema.EnvironmentVariable{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
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
		{
			desc:               "server side env vars with apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_NUMBER", Value: "54321", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_MULTILINE_FROM_SERVER_SIDE", Value: "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_QUOTED_SPECIAL", Value: "api_from_server_side.service.com::", Availability: []string{"RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_SINGLE_QUOTES", Value: "GOLANG is awesome!'", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
			},
			wantEnvMap: map[string]string{
				"SERVER_SIDE_ENV_VAR_NUMBER":                     "54321",
				"SERVER_SIDE_ENV_VAR_MULTILINE_FROM_SERVER_SIDE": "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
				"SERVER_SIDE_ENV_VAR_SPACED":                     "api3 - service -  com",
				"SERVER_SIDE_ENV_VAR_SINGLE_QUOTES":              "GOLANG is awesome!'",
				"SERVER_SIDE_ENV_VAR_DOUBLE_QUOTES":              "\"api4.service.com\"",
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
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_NUMBER", Value: "54321", Availability: []string{"BUILD", "RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_MULTILINE_FROM_SERVER_SIDE", Value: "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n", Availability: []string{"BUILD"}},
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_QUOTED_SPECIAL", Value: "api_from_server_side.service.com::", Availability: []string{"RUNTIME"}},
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD"}},
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_SINGLE_QUOTES", Value: "GOLANG is awesome!'", Availability: []string{"BUILD"}},
					apphostingschema.EnvironmentVariable{Variable: "SERVER_SIDE_ENV_VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc:               "server side env vars enabled but empty without apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			serverSideEnvVars:  []apphostingschema.EnvironmentVariable{},
			wantEnvMap:         map[string]string{},
			wantSchema:         apphostingschema.AppHostingSchema{},
		},
		{
			desc:               "server side env vars enabled but empty with apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars:  []apphostingschema.EnvironmentVariable{},
			wantEnvMap:         map[string]string{},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(3),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
			},
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
		// Convert server side env vars to string
		serverSideEnvVars := ""
		if test.serverSideEnvVars != nil {
			parsedServerSideEnvVars, err := json.Marshal(test.serverSideEnvVars)
			if err != nil {
				t.Errorf("Error in json marshalling serverSideEnvVars '%v'. Error was %v", test.serverSideEnvVars, err)
				return
			}
			serverSideEnvVars = string(parsedServerSideEnvVars)
		}
		opts := Options{
			SecretClient:                  fakeSecretClient,
			AppHostingYAMLPath:            test.appHostingYAMLPath,
			ProjectID:                     test.projectID,
			EnvironmentName:               test.environmentName,
			AppHostingYAMLOutputFilePath:  outputFilePathYAML,
			EnvDereferencedOutputFilePath: outputFilePathEnv,
			BackendRootDirectory:          "",
			BuildpackConfigOutputFilePath: outputFilePathBuildpackConfig,
			ServerSideEnvVars:             serverSideEnvVars,
		}

		if err := Prepare(context.Background(), opts); err != nil {
			t.Fatalf("Error in test '%v'. Error was %v", test.desc, err)
		}
		// Check dereferenced secret material env file
		actualEnvMapDereferenced, err := envvars.Read(outputFilePathEnv)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMap, actualEnvMapDereferenced); diff != "" {
			t.Errorf("Unexpected env map for test %v (-want, +got):\n%v", test.desc, diff)
		}

		// Check app hosting schema
		actualAppHostingSchema, err := apphostingschema.ReadAndValidateAppHostingSchemaFromFile(outputFilePathYAML)
		if err != nil {
			t.Errorf("reading in and validating apphosting.yaml at path %v: %v", outputFilePathYAML, err)
		}

		if diff := cmp.Diff(test.wantSchema, actualAppHostingSchema); diff != "" {
			t.Errorf("unexpected prepared YAML schema for test %q (-want, +got):\n%v", test.desc, diff)
		}
	}
}
