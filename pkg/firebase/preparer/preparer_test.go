package preparer

import (
	"testing"

	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

var (
	appHostingEnvPath string = testdata.MustGetPath("testdata/apphosting.env")
)

func TestPrepare(t *testing.T) {
	testDir := t.TempDir()
	outputFilePath := testDir + "/output"

	testCases := []struct {
		desc                  string
		appHostingEnvFilePath string
		projectID             string
		wantEnvMap            map[string]string
	}{
		{
			desc:                  "apphosting.env",
			appHostingEnvFilePath: appHostingEnvPath,
			projectID:             "test-project",
			wantEnvMap: map[string]string{
				"API_URL":           "api.service.com",
				"ENVIRONMENT":       "staging",
				"MULTILINE_ENV_VAR": "line 1\nline 2",
				"SECRET_API_KEY":    "projects/test-project/secrets/secretID/versions/11",
			},
		},
		{
			desc:                  "nonexistent apphosting.env",
			appHostingEnvFilePath: "",
			wantEnvMap:            map[string]string{},
		},
	}

	// Testing happy paths
	for _, test := range testCases {
		if err := Prepare(test.appHostingEnvFilePath, test.projectID, outputFilePath); err != nil {
			t.Errorf("Error in test '%v'. Error was %v", test.desc, err)
		}

		actualEnvMap, err := env.ReadEnv(outputFilePath)
		if err != nil {
			t.Errorf("Error reading in temp file: %v", err)
		}

		if diff := cmp.Diff(test.wantEnvMap, actualEnvMap); diff != "" {
			t.Errorf("Unexpected YAML for test %v (+got, -want):\n%v", test.desc, diff)
		}
	}
}
