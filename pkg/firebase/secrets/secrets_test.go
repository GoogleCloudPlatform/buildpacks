package secrets

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	}{
		{
			desc: "Pin secret values properly",
			inputEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": "projects/test-project/secrets/secretID/versions/5",
				"SECRET_FORMAT_LATEST": "projects/test-project/secrets/secretID/versions/latest",
			},
			wantEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": "projects/test-project/secrets/secretID/versions/5",
				"SECRET_FORMAT_LATEST": "projects/test-project/secrets/secretID/versions/latest",
			},
		},
	}

	for _, test := range testCases {
		err := PinVersionSecrets(test.inputEnvVars)

		if err != nil {
			t.Errorf("PinVersionSecrets(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
		}

		if diff := cmp.Diff(test.wantEnvVars, test.inputEnvVars); diff != "" {
			t.Errorf("unexpected pinned envVars for test %q (+got, -want):\n%v", test.desc, diff)
		}
	}
}

func TestDereferenceSecrets(t *testing.T) {
	testCases := []struct {
		desc         string
		inputEnvVars map[string]string
		wantEnvVars  map[string]string
	}{
		{
			desc: "Pin secret values properly",
			inputEnvVars: map[string]string{
				"API_URL":              "api.service.com",
				"SECRET_FORMAT_PINNED": "projects/test-project/secrets/secretID/versions/5",
				"SECRET_FORMAT_LATEST": "projects/test-project/secrets/secretID/versions/latest",
			},
			wantEnvVars: map[string]string{
				"API_URL":       "api.service.com",
				"FORMAT_PINNED": "secretString",
				"FORMAT_LATEST": "secretString",
			},
		},
	}

	for _, test := range testCases {
		gotEnvVars, err := DereferenceSecrets(test.inputEnvVars)

		if err != nil {
			t.Errorf("DereferenceSecrets(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
		}

		if diff := cmp.Diff(test.wantEnvVars, gotEnvVars); diff != "" {
			t.Errorf("unexpected dereferencing envVars for test %q (+got, -want):\n%v", test.desc, diff)
		}
	}
}
