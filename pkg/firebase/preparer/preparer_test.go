package preparer

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/fakesecretmanager"
	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	appHostingEnvPath string = testdata.MustGetPath("testdata/apphosting.env")
)

func TestPrepare(t *testing.T) {
	testDir := t.TempDir()
	outputFilePathReferenced := testDir + "/outputReferenced"
	outputFilePathDereferenced := testDir + "/outputDereferenced"

	testCases := []struct {
		desc                   string
		appHostingEnvFilePath  string
		projectID              string
		wantEnvMapReferenced   map[string]string
		wantEnvMapDereferenced map[string]string
	}{
		{
			desc:                  "apphosting.env",
			appHostingEnvFilePath: appHostingEnvPath,
			projectID:             "test-project",
			wantEnvMapReferenced: map[string]string{
				"API_URL":               "api.service.com",
				"ENVIRONMENT":           "staging",
				"MULTILINE_ENV_VAR":     "line 1\nline 2",
				"SECRET_API_KEY_LATEST": "projects/test-project/secrets/secretID/versions/12",
				"SECRET_API_KEY_PINNED": "projects/test-project/secrets/secretID/versions/11",
			},
			wantEnvMapDereferenced: map[string]string{
				"API_URL":           "api.service.com",
				"ENVIRONMENT":       "staging",
				"MULTILINE_ENV_VAR": "line 1\nline 2",
				"API_KEY_LATEST":    "secretString",
				"API_KEY_PINNED":    "secretString",
			},
		},
		{
			desc:                   "nonexistent apphosting.env",
			appHostingEnvFilePath:  "",
			wantEnvMapReferenced:   map[string]string{},
			wantEnvMapDereferenced: map[string]string{},
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		SecretVersionResponses: map[string]fakesecretmanager.GetSecretVersionResponse{
			"projects/test-project/secrets/secretID/versions/latest": fakesecretmanager.GetSecretVersionResponse{
				SecretVersion: &smpb.SecretVersion{
					Name:  "projects/test-project/secrets/secretID/versions/12",
					State: smpb.SecretVersion_ENABLED,
				},
			},
		},
	}

	// Testing happy paths
	for _, test := range testCases {
		if err := Prepare(context.Background(), fakeSecretClient, test.appHostingEnvFilePath, test.projectID, outputFilePathReferenced, outputFilePathDereferenced); err != nil {
			t.Errorf("Error in test '%v'. Error was %v", test.desc, err)
		}

		// Check referenced secret material env file
		actualEnvMapReferenced, err := env.ReadEnv(outputFilePathReferenced)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMapReferenced, actualEnvMapReferenced); diff != "" {
			t.Errorf("Unexpected YAML for test %v (+got, -want):\n%v", test.desc, diff)
		}

		// Check dereferenced secret material env file
		actualEnvMapDereferenced, err := env.ReadEnv(outputFilePathDereferenced)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMapDereferenced, actualEnvMapDereferenced); diff != "" {
			t.Errorf("Unexpected YAML for test %v (+got, -want):\n%v", test.desc, diff)
		}
	}
}
