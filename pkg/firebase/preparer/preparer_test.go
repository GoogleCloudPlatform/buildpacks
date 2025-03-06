package preparer

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/fakesecretmanager"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/envvars"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	"google3/third_party/golang/cmp/cmpopts/cmpopts"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google3/third_party/golang/protobuf/v2/proto/proto"
)

var (
	appHostingYAMLPath                 string = testdata.MustGetPath("testdata/apphosting.yaml")
	appHostingYAMLPathNonexistent      string = testdata.MustGetPath("testdata/nonexistent.yaml")
	appHostingYAMLConnectorIDPath      string = testdata.MustGetPath("testdata/apphosting.vpc-connector-id.yaml")
	appHostingYAMLConnectorNamePath    string = testdata.MustGetPath("testdata/apphosting.vpc-connector-name.yaml")
	appHostingYAMLNetworkIDPath        string = testdata.MustGetPath("testdata/apphosting.vpc-network-id.yaml")
	appHostingYAMLNetworkNamePath      string = testdata.MustGetPath("testdata/apphosting.vpc-network-name.yaml")
	apphostingStagingYAMLPath          string = testdata.MustGetPath("testdata/apphosting.staging.yaml")
	latestSecretName                   string = "projects/test-project/secrets/secretID/versions/12"
	pinnedSecretName                   string = "projects/test-project/secrets/secretID/versions/11"
	secretString                       string = "secretString"
	secretStringChecksum               int64  = int64(crc32.Checksum([]byte(secretString), crc32.MakeTable(crc32.Castagnoli)))
	serverProvidedFirebaseConfig       string = `{"databaseURL":"https://project-id-default-rtdb.firebaseio.com","projectId":"project-id","storageBucket":"project-id.firebasestorage.app"}`
	userProvidedFirebaseConfig         string = `{"databaseURL":"https://custom-user-database-rtdb.firebaseio.com","projectId":"project-id","storageBucket":"customStorageBucket.firebasestorage.app"}`
	serverProvidedFirebaseWebAppConfig string = `{"apiKey":"myApiKey","appId":"app:123","authDomain":"project-id.firebaseapp.com","databaseURL":"https://project-id-default-rtdb.firebaseio.com","messagingSenderId":"0123456789","projectId":"project-id","storageBucket":"project-id.firebasestorage.app"}`
	userProvidedFirebaseWebAppConfig   string = `{"apiKey":"myApiKey","appId":"app:123","authDomain":"project-id.firebaseapp.com","databaseURL":"https://custom-user-database-rtdb.firebaseio.com","messagingSenderId":"0123456789","projectId":"project-id","storageBucket":"customStorageBucket.firebasestorage.app"}`
)

func TestPrepare(t *testing.T) {
	testDir := t.TempDir()
	outputFilePathYAML := testDir + "/outputYAML"
	outputFilePathEnv := testDir + "/outputEnv"
	outputFilePathBuildpackConfig := testDir + "/outputBuildpackConfig"
	appHostingYAMLForPackPath := testDir + "/outputYAMLForPack"

	testCases := []struct {
		desc               string
		appHostingYAMLPath string
		projectID          string
		regionID           string
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
				"FIREBASE_CONFIG":         userProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG":  userProvidedFirebaseWebAppConfig,
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
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc:               "merges apphosting.<ENV>.yaml when apphosting.yaml not present",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			environmentName:    "staging",
			wantEnvMap: map[string]string{
				"API_URL":                 "api.staging.service.com",
				"STAGING_SECRET_VARIABLE": secretString,
				"FIREBASE_CONFIG":         serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG":  serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					MaxInstances: proto.Int32(2),
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc:               "merges and returns proper config even if apphostingYamlPath points to the env specific yaml file",
			appHostingYAMLPath: apphostingStagingYAMLPath,
			projectID:          "test-project",
			environmentName:    "staging",
			wantEnvMap: map[string]string{
				"API_URL":                 "api.staging.service.com",
				"STAGING_SECRET_VARIABLE": secretString,
				"FIREBASE_CONFIG":         serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG":  serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(512),
					MaxInstances: proto.Int32(2),
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc:               "non-existent apphosting.yaml nor apphosting.<ENV>.yaml",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc:               "merges server side env vars with apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "API_URL", Value: "override.service.com", Availability: []string{"BUILD"}},
				{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}},
			},
			wantEnvMap: map[string]string{
				"API_URL":                "override.service.com", // The apphosting.yaml value is 'api.service.com'.
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
				"VAR_NUMBER":             "12345",
				"FIREBASE_CONFIG":        userProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": userProvidedFirebaseWebAppConfig,
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
					{Variable: "API_URL", Value: "override.service.com", Availability: []string{"BUILD"}},             // This var is overridden from serverSideEnvVars.
					{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}}, // This var is only defined server-side.
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc:               "server side env vars without apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}}, // This var is only defined server-side.
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc:               "server side env vars enabled and empty without apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			serverSideEnvVars:  []apphostingschema.EnvironmentVariable{},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
		},
		{
			desc:               "server side env vars enabled and empty with apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars:  []apphostingschema.EnvironmentVariable{},
			wantEnvMap: map[string]string{
				"API_URL":                "api.service.com",
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\\nApt. 17\\nNew York, NY 10019\\n",
				"VAR_NUMBER":             "12345",
				"FIREBASE_CONFIG":        userProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": userProvidedFirebaseWebAppConfig,
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
					{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD"}},
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}},
				},
			},
		},
		{
			desc:               "noops vpc connector names",
			appHostingYAMLPath: appHostingYAMLConnectorNamePath,
			projectID:          "test-project",
			regionID:           "us-central1",
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						Connector: "projects/test-project/locations/us-central1/connectors/my-connector",
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
		},
		{
			desc:               "expands vpc connector ids",
			appHostingYAMLPath: appHostingYAMLConnectorIDPath,
			projectID:          "test-project",
			regionID:           "us-central1",
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						Connector: "projects/test-project/locations/us-central1/connectors/my-connector",
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
		},
		{
			desc:               "noops vpc network names",
			appHostingYAMLPath: appHostingYAMLNetworkNamePath,
			projectID:          "test-project",
			regionID:           "us-central1",
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						NetworkInterfaces: []apphostingschema.NetworkInterface{
							{
								Network:    "projects/test-project/global/networks/my-network",
								Subnetwork: "projects/test-project/regions/us-central1/subnetworks/my-subnet",
							},
						},
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
		},
		{
			desc:               "expands vpc network ids",
			appHostingYAMLPath: appHostingYAMLNetworkIDPath,
			projectID:          "test-project",
			regionID:           "us-central1",
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						NetworkInterfaces: []apphostingschema.NetworkInterface{
							{
								Network:    "projects/test-project/global/networks/my-network",
								Subnetwork: "projects/test-project/regions/us-central1/subnetworks/my-subnet",
							},
						},
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}},
				},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
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
			SecretClient:                      fakeSecretClient,
			AppHostingYAMLPath:                test.appHostingYAMLPath,
			ProjectID:                         test.projectID,
			Region:                            test.regionID,
			EnvironmentName:                   test.environmentName,
			AppHostingYAMLOutputFilePath:      outputFilePathYAML,
			EnvDereferencedOutputFilePath:     outputFilePathEnv,
			BackendRootDirectory:              "",
			BuildpackConfigOutputFilePath:     outputFilePathBuildpackConfig,
			FirebaseConfig:                    serverProvidedFirebaseConfig,
			FirebaseWebappConfig:              serverProvidedFirebaseWebAppConfig,
			ServerSideEnvVars:                 serverSideEnvVars,
			ApphostingPreprocessedPathForPack: appHostingYAMLForPackPath,
		}

		if err := Prepare(context.Background(), opts); err != nil {
			t.Fatalf("Error in test '%v'. Error was %v", test.desc, err)
		}
		// Check dereferenced secret material env file
		actualEnvMapDereferenced, err := envvars.Read(outputFilePathEnv)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMap, actualEnvMapDereferenced, cmpopts.SortMaps(func(a, b string) bool { return a < b })); diff != "" {
			t.Errorf("Unexpected env map for test %v (-want, +got):\n%v", test.desc, diff)
		}

		// Check app hosting schema
		actualAppHostingSchema, err := apphostingschema.ReadAndValidateFromFile(outputFilePathYAML)
		if err != nil {
			t.Errorf("reading in and validating apphosting.yaml at path %v: %v", outputFilePathYAML, err)
		}

		if diff := cmp.Diff(test.wantSchema, actualAppHostingSchema, cmpopts.SortSlices(func(a, b apphostingschema.EnvironmentVariable) bool { return a.Variable < b.Variable })); diff != "" {
			t.Errorf("unexpected prepared YAML schema for test %q (-want, +got):\n%v", test.desc, diff)
		}
	}
}
