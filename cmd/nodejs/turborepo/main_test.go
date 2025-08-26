package main

import (
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	bmd "github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		envs  []string
		want  int
	}{
		{
			name: "with_turbo_config",
			files: map[string]string{
				"index.js":   "",
				"turbo.json": "",
			},
			want: 0,
		},
		{
			name: "without_turbo_config",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, tc.envs, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name                string
		envs                []string
		files               map[string]string
		wantExitCode        int
		wantBuilderMetadata map[bmd.MetadataID]bmd.MetadataValue
	}{
		{
			name: "successfully_reads_turbo_json_and_app_package_json",
			envs: []string{"GOOGLE_BUILDABLE=apps/my-app"},
			files: map[string]string{
				"index.js": "",
				"turbo.json": `{
					"tasks": {
						"build": {
							"outputs": ["dist/**", ".next/**"],
							"cache": true
						}
					}
				}`,
				"apps/my-app/package.json": `{
					"name": "my-app"
				}`,
			},
			wantExitCode: 0,
			wantBuilderMetadata: map[bmd.MetadataID]bmd.MetadataValue{
				bmd.MonorepoName: bmd.MetadataValue("turbo"),
			},
		},
		{
			name: "ambiguous_application_name",
			files: map[string]string{
				"index.js": "",
				"turbo.json": `{
					"tasks": {
						"build": {
							"outputs": ["dist/**", ".next/**"],
							"cache": true
						}
					}
				}`,
				"apps/my-app/package.json": `{
					"name": "",
				}`,
			},
			wantExitCode: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			opts := []bpt.Option{
				bpt.WithTestName(tc.name),
				bpt.WithEnvs(tc.envs...),
				bpt.WithFiles(tc.files),
			}
			result, err := bpt.RunBuild(t, buildFn, opts...)
			if err != nil && tc.wantExitCode == 0 {
				t.Fatalf("error running build: %v, logs: %s", err, result.Output)
			}
			if result.ExitCode != tc.wantExitCode {
				t.Fatalf("build exit code mismatch, got: %d, want: %d", result.ExitCode, tc.wantExitCode)
			}
			for id, m := range tc.wantBuilderMetadata {
				if m != result.MetadataAdded()[id] {
					t.Errorf("builder metadata %q mismatch, got: %s, want: %s", bmd.MetadataIDNames[id], result.MetadataAdded()[id], m)
				}
			}
		})
	}
}
