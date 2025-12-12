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
	"github.com/buildpacks/libcnb/v2"
)

// IsUVRequirements checks if the application is a UV requirements.txt application.
func IsUVRequirements(ctx *gcp.Context) (bool, string, error) {
	if isPackageManagerConfigured(uv) {
		return true, fmt.Sprintf("environment variable %s is uv", env.PythonPackageManager), nil
	}
	if isPackageManagerEmpty() && isUVDefaultPackageManagerForRequirements(ctx) {
		return true, fmt.Sprintf("environment variable %s is not set, using uv as default package manager", env.PythonPackageManager), nil
	}
	return false, fmt.Sprintf("environment variable %s is not uv", env.PythonPackageManager), nil
}

// UVInstallRequirements installs dependencies from requirements.txt using 'uv pip install' and returns the path to the venv.
func UVInstallRequirements(ctx *gcp.Context, l *libcnb.Layer, reqs ...string) (string, error) {
	shouldInstall, err := prepareDependenciesLayer(ctx, l, "uv", reqs...)
	venvDir := filepath.Join(l.Path, ".venv")
	if err != nil {
		return "", fmt.Errorf("failed to prepare uv dependencies layer: %w", err)
	}
	if !shouldInstall {
		ctx.Logf("Dependencies are up to date, skipping installation.")
		return venvDir, nil
	}
	ctx.Logf("Installing application dependencies with uv.")

	pythonVersion, err := Version(ctx)
	if err != nil {
		return "", err
	}
	pythonVersion = strings.TrimPrefix(pythonVersion, "Python ")

	ctx.Logf("Creating virtual environment at %s with Python %s", venvDir, pythonVersion)
	venvCmd := []string{"uv", "venv", venvDir, "--python", pythonVersion}
	if _, err := ctx.Exec(venvCmd, gcp.WithUserAttribution); err != nil {
		return "", fmt.Errorf("failed to create virtual environment with uv: %w", err)
	}

	for _, req := range reqs {
		ctx.Logf("Installing dependencies from %s...", req)
		installCmd := []string{"uv", "pip", "install", "--requirement", req, "--reinstall", "--no-cache"}
		installCmd = appendVendoringFlags(installCmd)
		if _, err := ctx.Exec(installCmd, gcp.WithUserAttribution, gcp.WithEnv("VIRTUAL_ENV="+venvDir)); err != nil {
			return "", fmt.Errorf("failed to install dependencies from %s with uv: %w", req, err)
		}
	}
	ctx.Logf("Dependencies from requirements.txt installed to virtual environment at %s", venvDir)

	if err := compileBytecode(ctx, venvDir); err != nil {
		return "", fmt.Errorf("failed to compile bytecode: %w", err)
	}
	ctx.Logf("Finished compiling bytecode.")

	l.SharedEnvironment.Prepend("PATH", string(filepath.ListSeparator), filepath.Join(venvDir, "bin"))
	return venvDir, nil
}
