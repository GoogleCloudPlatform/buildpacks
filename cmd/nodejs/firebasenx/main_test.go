package main

import (
	"testing"

	bpt "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{
			name: "with nx config",
			files: map[string]string{
				"index.js": "",
				"nx.json":  "",
			},
			want: 0,
		},
		{
			name: "without nx config",
			files: map[string]string{
				"index.js": "",
			},
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bpt.TestDetect(t, detectFn, tc.name, tc.files, []string{}, tc.want)
		})
	}
}

func TestBuild(t *testing.T) {
	testCases := []struct {
		name         string
		envs         []string
		files        map[string]string
		wantExitCode int
	}{
		{
			name: "successfully read project.json",
			envs: []string{"GOOGLE_BUILDABLE=apps/my-project"},
			files: map[string]string{
				"index.js": "",
				"nx.json": `{
					"defaultProject": "my-project"
				}`,
				"apps/my-project/project.json": `{
					"name": "my-project",
					"targets": {
						"build": {
							"executor": "@angular-devkit/build-angular:application"
						}
					}
				}`,
			},
			wantExitCode: 0,
		},
		{
			name: "ambiguous project name",
			files: map[string]string{
				"index.js": "",
				"nx.json": `{
					"defaultProject": ""
				}`,
				"apps/my-project/project.json": `{
					"name": "my-project",
					"targets": {
						"build": {
							"executor": "@angular-devkit/build-angular:application"
						}
					}
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
		})
	}
}
