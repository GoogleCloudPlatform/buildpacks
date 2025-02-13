package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// helper function to create test files and directories
func createTestFiles(t *testing.T, testDir string, files []string) {
	t.Helper()
	for _, file := range files {
		fp := filepath.Join(testDir, file)
		err := os.MkdirAll(filepath.Dir(fp), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", filepath.Dir(fp), err)
		}
		err = os.WriteFile(fp, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", fp, err)
		}
	}
}

func TestDetectAppHostingYAMLPath(t *testing.T) {
	tests := []struct {
		name                 string
		workspacePath        string
		backendRootDirectory string
		want                 string
		wantError            bool
		files                []string
	}{
		{
			name:                 "improper_backend_root_directory_throws_error",
			workspacePath:        "/workspace",
			backendRootDirectory: "apps/my-app",
			wantError:            true,
		},
		{
			name:                 "returns_proper_path_to_apphosting_yaml",
			workspacePath:        "/workspace",
			backendRootDirectory: "",
			want:                 "/workspace/apphosting.yaml",
			files:                []string{"workspace/apphosting.yaml"},
		},
		{
			name:                 "returns_proper_path_to_apphosting_yaml_for_monorepo",
			workspacePath:        "/workspace",
			backendRootDirectory: "apps/my-app",
			want:                 "/workspace/apps/my-app/apphosting.yaml",
			files:                []string{"workspace/apps/my-app/apphosting.yaml"},
		},
		{
			name:                 "returns_root_apphosting_yaml_if_not_found_in_backend_root",
			workspacePath:        "/workspace",
			backendRootDirectory: "/apps/my-app",
			want:                 "/workspace/apphosting.yaml",
			files:                []string{"workspace/apphosting.yaml", "workspace/apps/my-app/test.txt"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir := t.TempDir()
			createTestFiles(t, testDir, test.files)

			workspaceAbsolutePath := filepath.Join(testDir, test.workspacePath)
			wantAbsolutePath := filepath.Join(testDir, test.want)
			got, err := DetectAppHostingYAMLPath(workspaceAbsolutePath, test.backendRootDirectory)
			if err != nil {
				if test.wantError {
					return
				}
				t.Fatalf("DetectAppHostingYAMLPath(workspacePath: %q, backendRootDirectory: %q) returned an unexpected error: %v", workspaceAbsolutePath, test.backendRootDirectory, err)
			}

			if got != wantAbsolutePath {
				t.Errorf("DetectAppHostingYAMLPath(workspacePath: %q, backendRootDirectory: %q) = %q, want: %q", workspaceAbsolutePath, test.backendRootDirectory, got, wantAbsolutePath)
			}
		})
	}
}

func TestDetectAppHostingYAMLRoot(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		want      string
		wantError bool
		files     []string
	}{
		{
			name:      "no_app_hosting_yaml",
			root:      "./apps/my-app",
			wantError: true,
		},
		{
			name:  "app_hosting_yaml_in_root",
			root:  "apps/my-app",
			want:  "apps",
			files: []string{"./apps/apphosting.yaml", "./apps/my-app/test.txt"},
		},
		{
			name:  "app_hosting_yaml_in_backend_root",
			root:  "apps/my-app",
			want:  "apps/my-app",
			files: []string{"apps/my-app/apphosting.yaml"},
		},
		{
			name:  "app_hosting_yaml_in_root_for_monorepo",
			root:  "apps/my-app",
			want:  "apps",
			files: []string{"apps/apphosting.yaml", "apps/my-app/test.txt"},
		},
		{
			name:  "environment_specific_app_hosting_yaml_in_root",
			root:  "apps/my-app",
			want:  "apps/my-app",
			files: []string{"apps/my-app/apphosting.staging.yaml"},
		},
		{
			name:  "environment_specific_app_hosting_yaml_in_monorepo",
			root:  "apps/my-app",
			want:  "apps",
			files: []string{"apps/apphosting.some_randome_env.yaml", "apps/my-app/test.txt"},
		},
		{
			name:  "apphosting_yaml_in_subdirectory_used_even_if_app_hosting_yaml_in_root",
			root:  "apps/my-app",
			want:  "apps/my-app",
			files: []string{"apps/my-app/apphosting.yaml", "apps/apphosting.yaml"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir := t.TempDir()
			createTestFiles(t, testDir, test.files)

			rootAbsolutePath := filepath.Join(testDir, test.root)
			wantAbsolutePath := filepath.Join(testDir, test.want)
			got, err := detectAppHostingYAMLRoot(rootAbsolutePath)
			if err != nil {
				if test.wantError {
					return
				}
				t.Fatalf("DetectAppHostingYAMLRoot(%q) returned an unexpected error: %v", rootAbsolutePath, err)
			}

			if got != wantAbsolutePath {
				t.Errorf("DetectAppHostingYAMLRoot(%q) = %q, want: %q", rootAbsolutePath, got, wantAbsolutePath)
			}
		})
	}
}
