package acceptance_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func TestAcceptance(t *testing.T) {
	imageCtx, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			Name:    "build a basic flex dotnet app",
			App:     "helloworld",
			MustUse: []string{flexBuildpack},
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, []string{
			"X_GOOGLE_TARGET_PLATFORM=flex",
			"GAE_APPLICATION_YAML_PATH=app.yaml",
		}...)

		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			acceptance.TestApp(t, imageCtx, tc)
		})
	}
}
