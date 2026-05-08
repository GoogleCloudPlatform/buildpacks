package preparer

import (
	"context"
	"encoding/json"
	"hash/crc32"
	"path/filepath"
	"strings"
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
	appHostingYAMLNgPath               string = testdata.MustGetPath("testdata/apphosting.ng.yaml")
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
	testCases := []struct {
		desc               string
		appHostingYAMLPath string
		projectID          string
		regionID           string
		environmentName    string
		serverSideEnvVars  []apphostingschema.EnvironmentVariable
		wantEnvMap         map[string]string
		wantSchema         apphostingschema.AppHostingSchema
		wantErr            string
	}{
		{
			desc:               "properly prepare apphosting.yaml for lifecycle builds",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			environmentName:    "staging",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"API_URL":                 "api.staging.service.com",
				"VAR_QUOTED_SPECIAL":      "api2.service.com::",
				"VAR_SPACED":              "api3 - service -  com",
				"VAR_SINGLE_QUOTES":       "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":       "\"api4.service.com\"",
				"MULTILINE_VAR":           "211 Broadway\nApt. 17\nNew York, NY 10019\n",
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
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "merges apphosting.<ENV>.yaml when apphosting.yaml not present",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			environmentName:    "staging",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
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
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "merges and returns proper config even if apphostingYamlPath points to the env specific yaml file",
			appHostingYAMLPath: apphostingStagingYAMLPath,
			projectID:          "test-project",
			environmentName:    "staging",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
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
					{Variable: "API_URL", Value: "api.staging.service.com", Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "STAGING_SECRET_VARIABLE", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: "apphosting.staging.yaml"},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "non-existent apphosting.yaml nor apphosting.<ENV>.yaml",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
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
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"API_URL":                "override.service.com", // The apphosting.yaml value is 'api.service.com'.
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\nApt. 17\nNew York, NY 10019\n",
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
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "API_URL", Value: "override.service.com", Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseConsole},             // This var is overridden from serverSideEnvVars.
					{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole}, // This var is only defined server-side.
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "writes env vars for lifecycle builds if outputFilePathEnv starts with /platform/env",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "API_URL", Value: "override.service.com", Availability: []string{"BUILD"}},
				{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}},
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"API_URL":                "override.service.com", // The apphosting.yaml value is 'api.service.com'.
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\nApt. 17\nNew York, NY 10019\n",
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
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "API_URL", Value: "override.service.com", Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseConsole},             // This var is overridden from serverSideEnvVars.
					{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole}, // This var is only defined server-side.
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "server side env vars without apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}},
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "SERVER_SIDE_VAR", Value: "I'm a server side var!", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole}, // This var is only defined server-side.
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "server side env vars enabled and contains only system variables without apphosting.yaml",
			appHostingYAMLPath: "",
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "server side env vars enabled and contains only system variables with apphosting.yaml",
			appHostingYAMLPath: appHostingYAMLPath,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"API_URL":                "api.service.com",
				"VAR_QUOTED_SPECIAL":     "api2.service.com::",
				"VAR_SPACED":             "api3 - service -  com",
				"VAR_SINGLE_QUOTES":      "I said, 'I'm learning YAML!'",
				"VAR_DOUBLE_QUOTES":      "\"api4.service.com\"",
				"MULTILINE_VAR":          "211 Broadway\nApt. 17\nNew York, NY 10019\n",
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
					CPU:          proto.Float32(1),
					MemoryMiB:    proto.Int32(1024),
					Concurrency:  proto.Int32(100),
					MaxInstances: proto.Int32(4),
					MinInstances: proto.Int32(0),
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_QUOTED_SPECIAL", Value: "api2.service.com::", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SPACED", Value: "api3 - service -  com", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_SINGLE_QUOTES", Value: "I said, 'I'm learning YAML!'", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_DOUBLE_QUOTES", Value: "\"api4.service.com\"", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "MULTILINE_VAR", Value: "211 Broadway\nApt. 17\nNew York, NY 10019\n", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VAR_NUMBER", Value: "12345", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_CONFIG", Value: userProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: userProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "API_KEY", Secret: latestSecretName, Availability: []string{"BUILD"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "VERBOSE_API_KEY", Secret: latestSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "PINNED_VERBOSE_API_KEY", Secret: pinnedSecretName, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "noops vpc connector names",
			appHostingYAMLPath: appHostingYAMLConnectorNamePath,
			projectID:          "test-project",
			regionID:           "us-central1",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						Connector: "projects/test-project/locations/us-central1/connectors/my-connector",
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
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
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantSchema: apphostingschema.AppHostingSchema{
				RunConfig: apphostingschema.RunConfig{
					VpcAccess: &apphostingschema.VpcAccess{
						Connector: "projects/test-project/locations/us-central1/connectors/my-connector",
					},
				},
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
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
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
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
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
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
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
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
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
		},
		{
			desc:      "accepts correct user provided NG_TRUST_PROXY_HEADERS",
			projectID: "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}},
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:      "errors if user provided NG_TRUST_PROXY_HEADERS lacks RUNTIME availability",
			projectID: "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"BUILD"}},
			},
			wantErr: "user-defined environment variable NG_TRUST_PROXY_HEADERS must include RUNTIME in its availability",
		},
		{
			desc:      "adds default NG_TRUST_PROXY_HEADERS when not provided",
			projectID: "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:      "user provided NG_TRUST_PROXY_HEADERS without availability defaults to runtime and build",
			projectID: "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host"},
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "default.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
				"NG_TRUST_PROXY_HEADERS": "X-Forwarded-Host",
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole},
					{Variable: "NG_ALLOWED_HOSTS", Value: "default.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:      "errors on incorrect user provided NG_TRUST_PROXY_HEADERS",
			projectID: "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_TRUST_PROXY_HEADERS", Value: "Invalid-Header"},
			},
			wantErr: `invalid value for NG_TRUST_PROXY_HEADERS (Angular trust proxy headers): got "Invalid-Header", want "X-Forwarded-Host"`,
		},
		{
			desc:               "derives NG_ALLOWED_HOSTS from X_FIREBASE_SUPPORTED_HOSTS",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "example.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "NG_ALLOWED_HOSTS", Value: "example.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "does not override server-side user-defined NG_ALLOWED_HOSTS with X_FIREBASE_SUPPORTED_HOSTS",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_ALLOWED_HOSTS", Value: "user-defined.com", Availability: []string{"RUNTIME"}},
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "system-defined.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "NG_ALLOWED_HOSTS", Value: "user-defined.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "does not override user-defined NG_ALLOWED_HOSTS in apphosting.yaml with X_FIREBASE_SUPPORTED_HOSTS",
			appHostingYAMLPath: appHostingYAMLNgPath,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: "system-defined.com"},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "NG_ALLOWED_HOSTS", Value: "yaml-defined.com", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceAppHostingYAML},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "succeeds if NG_ALLOWED_HOSTS is missing and X_FIREBASE_SUPPORTED_HOSTS is empty",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "X_FIREBASE_SUPPORTED_HOSTS", Value: ""},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "succeeds if NG_ALLOWED_HOSTS is missing and X_FIREBASE_SUPPORTED_HOSTS is missing",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars:  []apphostingschema.EnvironmentVariable{},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
				},
			},
		},
		{
			desc:               "fails if user-provided NG_ALLOWED_HOSTS has no RUNTIME availability",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_ALLOWED_HOSTS", Value: "example.com", Availability: []string{"BUILD"}},
			},
			wantErr: "NG_ALLOWED_HOSTS environment variable must be set with RUNTIME availability",
		},
		{
			desc:               "user-defined NG_ALLOWED_HOSTS with both BUILD and RUNTIME availability",
			appHostingYAMLPath: appHostingYAMLPathNonexistent,
			projectID:          "test-project",
			serverSideEnvVars: []apphostingschema.EnvironmentVariable{
				{Variable: "NG_ALLOWED_HOSTS", Value: "both.com", Availability: []string{"BUILD", "RUNTIME"}},
			},
			wantEnvMap: map[string]string{
				"FIREBASE_CONFIG":        serverProvidedFirebaseConfig,
				"FIREBASE_WEBAPP_CONFIG": serverProvidedFirebaseWebAppConfig,
				"NG_ALLOWED_HOSTS":       "both.com",
			},
			wantSchema: apphostingschema.AppHostingSchema{
				Env: []apphostingschema.EnvironmentVariable{
					{Variable: "NG_ALLOWED_HOSTS", Value: "both.com", Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseConsole},
					{Variable: "FIREBASE_CONFIG", Value: serverProvidedFirebaseConfig, Availability: []string{"BUILD", "RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "FIREBASE_WEBAPP_CONFIG", Value: serverProvidedFirebaseWebAppConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem},
					{Variable: "NG_TRUST_PROXY_HEADERS", Value: "X-Forwarded-Host", Availability: []string{"RUNTIME"}, Source: apphostingschema.SourceFirebaseSystem},
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
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			testDir := t.TempDir()
			outputFilePathYAML := filepath.Join(testDir, "outputYAML")
			outputFilePathBuildpackConfig := filepath.Join(testDir, "outputBuildpackConfig")
			appHostingYAMLForPackPath := filepath.Join(testDir, "outputYAMLForPack")
			envOutputPath := filepath.Join(testDir, "outputEnv")

			// Convert server side env vars to string
			serverSideEnvVars := ""
			if test.serverSideEnvVars != nil {
				parsedServerSideEnvVars, err := json.Marshal(test.serverSideEnvVars)
				if err != nil {
					t.Fatalf("Error in json marshalling serverSideEnvVars '%v'. Error was %v", test.serverSideEnvVars, err)
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
				EnvDereferencedOutputFilePath:     envOutputPath,
				BackendRootDirectory:              "",
				BuildpackConfigOutputFilePath:     outputFilePathBuildpackConfig,
				FirebaseConfig:                    serverProvidedFirebaseConfig,
				FirebaseWebappConfig:              serverProvidedFirebaseWebAppConfig,
				ServerSideEnvVars:                 serverSideEnvVars,
				ApphostingPreprocessedPathForPack: appHostingYAMLForPackPath,
			}

			err := Prepare(context.Background(), opts)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("Expected error containing %q, got %v", test.wantErr, err)
				}
				return // Success for error case
			}
			if err != nil {
				t.Fatalf("Unexpected error in test '%v': %v", test.desc, err)
			}

			var actualEnvMapDereferenced map[string]string
			actualEnvMapDereferenced, err = envvars.ReadLifecycle(envOutputPath)
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
		})
	}
}
