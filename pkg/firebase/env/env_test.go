package env

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSanitizeAppHostingEnvFields(t *testing.T) {
	testCases := []struct {
		desc         string
		inputEnvVars map[string]string
		wantEnvVars  map[string]string
	}{
		{
			desc: "Remove no keys when all env vars are valid",
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
			desc: "Remove keys containing reserved Cloud Run env var keys",
			inputEnvVars: map[string]string{
				"API_URL":   "api.service.com",
				"K_SERVICE": "k.service.com",
			},
			wantEnvVars: map[string]string{
				"API_URL": "api.service.com",
			},
		},
		{
			desc: "Remove keys containing reserved Firebase env vars keys",
			inputEnvVars: map[string]string{
				"API_URL":     "api.service.com",
				"FIREBASE_ID": "firebaseId",
			},
			wantEnvVars: map[string]string{
				"API_URL": "api.service.com",
			},
		},
	}

	for _, test := range testCases {
		gotEnvVars, err := SanitizeAppHostingEnv(test.inputEnvVars)
		if err != nil {
			t.Errorf("SanitizeAppHostingEnv(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
		}
		if diff := cmp.Diff(test.wantEnvVars, gotEnvVars); diff != "" {
			t.Errorf("unexpected sanitized envVars for test %q (+got, -want):\n%v", test.desc, diff)
		}
	}
}

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
		gotEnvVars, err := NormalizeAppHostingSecretsEnv(test.inputEnvVars, test.projectID)

		// Happy Case
		if test.wantErr == "" {
			if err != nil {
				t.Errorf("NormalizeAppHostingSecretsEnv(%q) = %v, want %v", test.desc, err, test.wantEnvVars)
			}

			if diff := cmp.Diff(test.wantEnvVars, gotEnvVars); diff != "" {
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
