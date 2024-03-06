package secrets

import (
	"context"
	"hash/crc32"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/fakesecretmanager"
	"github.com/google/go-cmp/cmp"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	ctx                  context.Context = context.Background()
	pinnedSecretName     string          = "projects/test-project/secrets/secretID/versions/5"
	latestSecretName     string          = "projects/test-project/secrets/secretID/versions/latest"
	secretString         string          = "secretString"
	secretStringChecksum int64           = int64(crc32.Checksum([]byte(secretString), crc32.MakeTable(crc32.Castagnoli)))
)

func TestNormalizeAppHostingSecretsEnv(t *testing.T) {
	testCases := []struct {
		desc         string
		projectID    string
		inputEnvVars map[string]string
		wantEnvVars  map[string]string
		wantErr      string
	}{
		{
			desc:      "Normalize different secret formats properly",
			projectID: "test-project",
			inputEnvVars: map[string]string{
				"API_URL":             "api.service.com",
				"SECRET_FORMAT_ONE":   "secretID@5",
				"SECRET_FORMAT_TWO":   "secretID",
				"SECRET_FORMAT_THREE": "@6",
				"SECRET_FORMAT_FOUR":  "",
			},
			wantEnvVars: map[string]string{
				"API_URL":             "api.service.com",
				"SECRET_FORMAT_ONE":   "projects/test-project/secrets/secretID/versions/5",
				"SECRET_FORMAT_TWO":   "projects/test-project/secrets/secretID/versions/latest",
				"SECRET_FORMAT_THREE": "projects/test-project/secrets/FORMAT_THREE/versions/6",
				"SECRET_FORMAT_FOUR":  "projects/test-project/secrets/FORMAT_FOUR/versions/latest",
			},
		},
		{
			desc:      "Normalize nothing when there are no secrets",
			projectID: "test-project",
			inputEnvVars: map[string]string{
				"API_URL":     "api.service.com",
				"ENVIRONMENT": "staging",
			},
			wantEnvVars: map[string]string{
				"API_URL":     "api.service.com",
				"ENVIRONMENT": "staging",
			},
		},
		{
			desc:      "Throw an error",
			projectID: "test-project",
			inputEnvVars: map[string]string{
				"API_URL":           "api.service.com",
				"SECRET_FORMAT_ONE": "secretID@@@5", // Invalid format
			},
			wantErr: "invalid secret format",
		},
	}

	for _, test := range testCases {
		err := NormalizeAppHostingSecretsEnv(test.inputEnvVars, test.projectID)

		// Happy Case
		if test.wantErr == "" {
			if err != nil {
				t.Errorf("NormalizeAppHostingSecretsEnv(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
			}

			if diff := cmp.Diff(test.wantEnvVars, test.inputEnvVars); diff != "" {
				t.Errorf("unexpected normalized envVars for test %q (+got, -want):\n%v", test.desc, diff)
			}
		} else {
			// Error Case
			if err == nil {
				t.Errorf("calling NormalizeAppHostingSecretsEnv did not produce an error for test %q", test.desc)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error not in expected format for test %q.\nGot: %v\nWant: %v", test.desc, err, test.wantErr)
			}
		}
	}
}

func TestPinVersionSecrets(t *testing.T) {
	testCases := []struct {
		desc         string
		inputEnvVars map[string]string
		wantEnvVars  map[string]string
		wantErr      bool
	}{
		{
			desc: "Pin secret values properly",
			inputEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": pinnedSecretName,
				"SECRET_FORMAT_LATEST": latestSecretName,
			},
			wantEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": pinnedSecretName,
				"SECRET_FORMAT_LATEST": pinnedSecretName, // Latest secret must become pinned
			},
		},
		{
			desc: "Throw an error when secret version is not found",
			inputEnvVars: map[string]string{
				"API_URL":                      "api.service.com",
				"SECRET_FORMAT_PINNED":         pinnedSecretName,
				"SECRET_FORMAT_LATEST_INVALID": "projects/test-project/secrets/invalidSecretID/versions/latest",
			},
			wantErr: true,
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		SecretVersionResponses: map[string]fakesecretmanager.GetSecretVersionResponse{
			latestSecretName: fakesecretmanager.GetSecretVersionResponse{
				SecretVersion: &smpb.SecretVersion{
					Name:  pinnedSecretName,
					State: smpb.SecretVersion_ENABLED,
				},
			},
		},
	}

	for _, test := range testCases {
		err := PinVersionSecrets(ctx, fakeSecretClient, test.inputEnvVars)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("PinVersionSecrets(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
			}

			if diff := cmp.Diff(test.wantEnvVars, test.inputEnvVars); diff != "" {
				t.Errorf("unexpected pinned envVars for test %q (+got, -want):\n%v", test.desc, diff)
			}
			// Error Path
		} else {
			if err == nil {
				t.Errorf("PinVersionSecrets(%q) = %v, want error", test.desc, err)
			}
		}
	}
}

func TestDereferenceSecrets(t *testing.T) {
	testCases := []struct {
		desc         string
		inputEnvVars map[string]string
		wantEnvVars  map[string]string
		wantErr      bool
	}{
		{
			desc: "Dereference secret values properly",
			inputEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": pinnedSecretName,
			},
			wantEnvVars: map[string]string{
				"API_URL":       "api.service.com",
				"FORMAT_PINNED": secretString,
			},
		},
		{
			desc: "Throw an error when secret version is not found",
			inputEnvVars: map[string]string{
				"API_URL":                      "api.service.com",
				"SECRET_FORMAT_PINNED":         pinnedSecretName,
				"SECRET_FORMAT_LATEST_INVALID": "projects/test-project/secrets/invalidSecretID/versions/latest",
			},
			wantErr: true,
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		AccessSecretVersionResponses: map[string]fakesecretmanager.AccessSecretVersionResponse{
			pinnedSecretName: fakesecretmanager.AccessSecretVersionResponse{
				Response: &smpb.AccessSecretVersionResponse{
					Payload: &smpb.SecretPayload{
						Data:       []byte(secretString),
						DataCrc32C: &secretStringChecksum,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		gotEnvVars, err := DereferenceSecrets(ctx, fakeSecretClient, test.inputEnvVars)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("unexpected error for DereferenceSecrets(%q): %v", test.desc, err)
			}

			if diff := cmp.Diff(test.wantEnvVars, gotEnvVars); diff != "" {
				t.Errorf("unexpected dereferenced secrets for test %q (+got, -want):\n%v", test.desc, diff)
			}
		} else {
			if err == nil {
				t.Errorf("DereferenceSecrets(%q) = %v, want error", test.desc, err)
			}
		}
	}
}
