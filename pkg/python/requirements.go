// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// IsUVRequirementsEnabled checks if the uv requirements.txt feature is enabled.
// For any future changes to the release stage, this is the single place to make changes.
func IsUVRequirementsEnabled(ctx *gcp.Context) bool {
	return env.IsAlphaSupported()
}

// IsUVRequirements checks if the application is a UV requirements.txt application.
func IsUVRequirements(ctx *gcp.Context) (bool, string, error) {
	requirementsExists, err := ctx.FileExists(requirements)
	if err != nil {
		return false, "", fmt.Errorf("checking for %s: %w", requirements, err)
	}
	if !requirementsExists {
		return false, fmt.Sprintf("%s not found", requirements), nil
	}
	if isPackageManagerConfigured(uv) {
		return true, fmt.Sprintf("%s found and environment variable %s is uv", requirements, env.PythonPackageManager), nil
	}
	if isPackageManagerEmpty() && isUVDefaultPackageManagerForRequirements(ctx) {
		return true, fmt.Sprintf("%s found and %s is not set, using uv as default package manager", requirements, env.PythonPackageManager), nil
	}
	return false, fmt.Sprintf("%s found but environment variable %s is not uv", requirements, env.PythonPackageManager), nil
}

// UVInstallRequirements installs dependencies from requirements.txt using 'uv pip install'.
func UVInstallRequirements(ctx *gcp.Context) error {
	layer, err := ctx.Layer(uvDepsLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", uvDepsLayer, err)
	}

	pythonVersion, err := Version(ctx)
	if err != nil {
		return err
	}
	pythonVersion = strings.TrimPrefix(pythonVersion, "Python ")

	venvDir := filepath.Join(layer.Path, ".venv")
	ctx.Logf("Creating virtual environment at %s with Python %s", venvDir, pythonVersion)
	venvCmd := []string{"uv", "venv", venvDir, "--python", pythonVersion}
	if _, err := ctx.Exec(venvCmd, gcp.WithUserAttribution); err != nil {
		return fmt.Errorf("failed to create virtual environment with uv: %w", err)
	}

	ctx.Logf("Installing dependencies with `uv pip install -r requirements.txt` into the virtual environment...")
	installCmd := []string{"uv", "pip", "install", "-r", "requirements.txt"}
	if _, err := ctx.Exec(installCmd, gcp.WithUserAttribution, gcp.WithEnv("VIRTUAL_ENV="+venvDir)); err != nil {
		return fmt.Errorf("failed to install requirements.txt with uv: %w", err)
	}
	ctx.Logf("Dependencies from requirements.txt installed to virtual environment at %s", venvDir)

	venvBinDir := filepath.Join(venvDir, "bin")
	layer.SharedEnvironment.Prepend("PATH", string(filepath.ListSeparator), venvBinDir)
	return nil
}
