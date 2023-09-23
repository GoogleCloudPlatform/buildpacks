package acceptance

import (
	"strings"
	"testing"
)

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		name               string
		srcDir             string
		image              string
		builderName        string
		runName            string
		env                map[string]string
		cache              bool
		mustContainArgs    []string
		mustNotContainArgs []string
	}{
		{
			name:        "generic run images do not skip adding runtime as launch layer",
			srcDir:      "some/src/dir",
			image:       "my-image",
			builderName: "gcr.io/my-builder",
			runName:     "gcr.io/gae-runtimes/buildpacks/google-gae-22/nodejs/run",
			mustNotContainArgs: []string{
				"--env X_GOOGLE_SKIP_RUNTIME_LAUNCH",
			},
		},
		{
			name:        "non-generic run images skip adding runtime as launch layer",
			srcDir:      "some/src/dir",
			image:       "my-image",
			builderName: "gcr.io/my-builder",
			runName:     "gcr.io/gae-runtimes/buildpacks/nodejs14/run",
			mustContainArgs: []string{
				"--env X_GOOGLE_SKIP_RUNTIME_LAUNCH=true",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := buildCommand(tc.srcDir, tc.image, tc.builderName, tc.runName, tc.env, tc.cache)
			got := strings.Join(args, " ")
			for _, want := range tc.mustContainArgs {
				if !strings.Contains(got, want) {
					t.Errorf("buildCommand(%q, %q, %q, %q, %v, %v) = %q, must contain %q", tc.srcDir, tc.image, tc.builderName, tc.runName, tc.env, tc.cache, got, tc.mustContainArgs)
				}
			}

			for _, doNotWant := range tc.mustNotContainArgs {
				if strings.Contains(got, doNotWant) {
					t.Errorf("buildCommand(%q, %q, %q, %q, %v, %v) = %q, must not contain %q", tc.srcDir, tc.image, tc.builderName, tc.runName, tc.env, tc.cache, got, tc.mustNotContainArgs)
				}
			}
		})
	}
}

func TestHasRuntimePreinstalled(t *testing.T) {
	testCases := []struct {
		name                   string
		image                  string
		hasRuntimePreinstalled bool
	}{
		{
			name:                   "nonGenericRunImage",
			image:                  "gcr.io/gae-runtimes/buildpacks/nodejs14/run",
			hasRuntimePreinstalled: true,
		},
		{
			name:                   "tagDoesntMatter",
			image:                  "gcr.io/gae-runtimes/buildpacks/nodejs14/run:tagdoesntmatter",
			hasRuntimePreinstalled: true,
		},
		{
			name:                   "genericRunImage",
			image:                  "gcr.io/gae-runtimes/buildpacks/google-gae-22/nodejs/run",
			hasRuntimePreinstalled: false,
		},
		{
			name:                   "someCustomImage",
			image:                  "us.gcr.io/some-random/custom/image:latest",
			hasRuntimePreinstalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.image, func(t *testing.T) {
			got := hasRuntimePreinstalled(tc.image)
			if got != tc.hasRuntimePreinstalled {
				t.Errorf("hasRuntimePreinstalled(%q) = %v, want %v", tc.image, got, tc.hasRuntimePreinstalled)
			}
		})
	}
}
