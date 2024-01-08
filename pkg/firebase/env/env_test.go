package env

import (
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
