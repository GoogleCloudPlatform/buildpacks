package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteBuildDirectoryContext(t *testing.T) {
	testCases := []struct {
		name                         string
		appDirectoryPath             string
		workingDirectory             string
		files                        []string
		wantBuildDirectory           string
		wantRelativeProjectDirectory string
		wantError                    bool
	}{
		{
			name:                         "no_app_directory_path",
			appDirectoryPath:             "",
			wantBuildDirectory:           "",
			wantRelativeProjectDirectory: "",
		},
		{
			name:                         "nx_monorepo",
			appDirectoryPath:             "apps/my-app",
			wantBuildDirectory:           ".",
			wantRelativeProjectDirectory: "apps/my-app",
			files:                        []string{"apps/my-app/project.json", "nx.json"},
		},
		{
			name:                         "turborepo",
			appDirectoryPath:             "apps/my-app",
			wantBuildDirectory:           ".",
			wantRelativeProjectDirectory: "apps/my-app",
			files:                        []string{"apps/my-app/package.json", "turbo.json"},
		},
		{
			name:                         "invalid_monorepo",
			appDirectoryPath:             "apps/my-app",
			wantBuildDirectory:           "apps/my-app",
			wantRelativeProjectDirectory: "",
			files:                        []string{"apps/my-app/package.json", "monorepo.json"},
		},
		{
			name:                         "subdirectory",
			appDirectoryPath:             "frontend",
			wantBuildDirectory:           "frontend",
			wantRelativeProjectDirectory: "",
			files:                        []string{"frontend/package.json"},
		},
		{
			name:                         "nx_monorepo_in_subdirectory",
			appDirectoryPath:             "nx/apps/my-app",
			wantBuildDirectory:           "nx",
			wantRelativeProjectDirectory: "apps/my-app",
			files:                        []string{"nx/nx.json", "nx/apps/my-app/project.json"},
		},
		{
			name:                         "turborepo_in_subdirectory",
			appDirectoryPath:             "turbo/apps/my-app",
			wantBuildDirectory:           "turbo",
			wantRelativeProjectDirectory: "apps/my-app",
			files:                        []string{"turbo/turbo.json", "turbo/apps/my-app/package.json"},
		},
		{
			name:                         "invalid_app_directory_path",
			appDirectoryPath:             "path/to/nowhere",
			wantBuildDirectory:           "",
			wantRelativeProjectDirectory: "",
			files:                        []string{},
			wantError:                    true,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			testDir := t.TempDir()
			for _, file := range test.files {
				fp := filepath.Join(testDir, file)
				err := os.MkdirAll(filepath.Dir(fp), 0755)
				if err != nil {
					t.Fatalf("failed to create directory %s: %v", filepath.Dir(fp), err)
				}
				err = os.WriteFile(filepath.Join(testDir, file), []byte(""), 0644)
				if err != nil {
					t.Fatalf("failed to write file %s: %v", fp, err)
				}
			}

			buildpackConfigFilePath := filepath.Join(testDir, "tmp")
			err := WriteBuildDirectoryContext(testDir, test.appDirectoryPath, buildpackConfigFilePath)
			if err != nil {
				if test.wantError {
					return
				}
				t.Errorf("WriteBuildDirectoryContext(%v, %v, %v) failed unexpectedly; err = %v", testDir, test.appDirectoryPath, buildpackConfigFilePath, err)
			}

			gotRelativeProjectDirectory, err := os.ReadFile(filepath.Join(buildpackConfigFilePath, "relative-project-directory.txt"))
			if err != nil {
				t.Errorf("error reading in build directory file: %v", err)
			}
			if string(gotRelativeProjectDirectory) != test.wantRelativeProjectDirectory {
				t.Errorf("got %v, want %v", string(gotRelativeProjectDirectory), test.wantRelativeProjectDirectory)
			}

			gotBuildDirectory, err := os.ReadFile(filepath.Join(buildpackConfigFilePath, "build-directory.txt"))
			if err != nil {
				t.Errorf("error reading in build directory file: %v", err)
			}
			if string(gotBuildDirectory) != test.wantBuildDirectory {
				t.Errorf("got %v, want %v", string(gotBuildDirectory), test.wantBuildDirectory)
			}
		})

	}
}
