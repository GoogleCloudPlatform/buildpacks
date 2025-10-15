// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package python

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestIsUVRequirements(t *testing.T) {
	testCases := []struct {
		name    string
		files   map[string]string
		envVars map[string]string
		want    bool
		wantMsg string
	}{
		{
			name:    "should_be_true_with_requirements.txt_and_uv_env_var",
			files:   map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{env.PythonPackageManager: "uv"},
			want:    true,
			wantMsg: fmt.Sprintf("%s found and environment variable %s is uv", requirements, env.PythonPackageManager),
		},
		{
			name:    "should_be_true_with_requirements.txt_and_case-insensitive_UV_env_var",
			files:   map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{env.PythonPackageManager: "Uv"},
			want:    true,
			wantMsg: fmt.Sprintf("%s found and environment variable %s is uv", requirements, env.PythonPackageManager),
		},
		{
			name:    "should_be_false_without_requirements.txt",
			files:   map[string]string{},
			envVars: map[string]string{env.PythonPackageManager: "uv"},
			want:    false,
			wantMsg: fmt.Sprintf("%s not found", requirements),
		},
		{
			name:    "should_be_false_when_env_var_is_pip",
			files:   map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{env.PythonPackageManager: "pip"},
			want:    false,
			wantMsg: fmt.Sprintf("%s found but environment variable %s is not uv", requirements, env.PythonPackageManager),
		},
		{
			name:  "should_be_false_when_env_var_is_not_set_and_py_lt_314",
			files: map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.13.9",
			},
			want:    false,
			wantMsg: fmt.Sprintf("%s found but environment variable %s is not uv", requirements, env.PythonPackageManager),
		},
		{
			name:  "should_be_true_when_env_var_is_not_set_and_py_eq_314",
			files: map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.14.0",
			},
			want:    true,
			wantMsg: fmt.Sprintf("%s found and %s is not set, using uv as default package manager", requirements, env.PythonPackageManager),
		},
		{
			name:  "should_be_true_when_env_var_is_not_set_and_py_gt_314",
			files: map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{
				"GOOGLE_RUNTIME_VERSION": "3.15.1",
			},
			want:    true,
			wantMsg: fmt.Sprintf("%s found and %s is not set, using uv as default package manager", requirements, env.PythonPackageManager),
		},
		{
			name:  "should_be_true_when_env_is_uv_and_py_lt_314",
			files: map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{
				env.PythonPackageManager: "uv",
				"GOOGLE_RUNTIME_VERSION": "3.13.9",
			},
			want:    true,
			wantMsg: fmt.Sprintf("%s found and environment variable %s is uv", requirements, env.PythonPackageManager),
		},
		{
			name:  "should_be_false_when_env_is_pip_and_py_gt_314",
			files: map[string]string{"requirements.txt": "flask"},
			envVars: map[string]string{
				env.PythonPackageManager: "pip",
				"GOOGLE_RUNTIME_VERSION": "3.14.0",
			},
			want:    false,
			wantMsg: fmt.Sprintf("%s found but environment variable %s is not uv", requirements, env.PythonPackageManager),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			appDir := setupTest(t, tc.files) // Assumes setupTest is available in this package

			ctx := gcp.NewContext(gcp.WithApplicationRoot(appDir))
			isUV, msg, err := IsUVRequirements(ctx)

			if err != nil {
				t.Fatalf("IsUVRequirements() got an unexpected error: %v", err)
			}
			if isUV != tc.want {
				t.Errorf("IsUVRequirements() = %v, want %v", isUV, tc.want)
			}
			if msg != tc.wantMsg {
				t.Errorf("IsUVRequirements() message = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}
