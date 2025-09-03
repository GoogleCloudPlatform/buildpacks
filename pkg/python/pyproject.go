// Copyright 2024 Google LLC
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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/BurntSushi/toml"
)

const (
	poetryLayer        = "poetry"
	dependenciesLayer  = "poetry-dependencies"
	pyprojectToml      = "pyproject.toml"
	poetryLock         = "poetry.lock"
	poetryVenvsPathEnv = "POETRY_VIRTUALENVS_PATH"
	uvLock             = "uv.lock"
	uvLayer            = "uv"
	uvDepsLayer        = "uv-dependencies"
)

var (
	poetryInstallCmd = []string{"poetry", "install", "--no-interaction", "--sync", "--only", "main", "--no-root"}
	poetryEnvInfoCmd = []string{"poetry", "env", "info", "--path"}
	poetryLockCmd    = []string{"poetry", "lock", "--no-interaction"}
	uvLockCmd        = []string{"uv", "lock"}
	uvSyncCmd        = []string{"uv", "sync", "--active"}
)

// IsPoetryProject checks if the application is a Poetry project.
func IsPoetryProject(ctx *gcp.Context) (bool, string, error) {
	poetryLockExists, err := ctx.FileExists(poetryLock)
	if err != nil {
		return false, "", fmt.Errorf("checking for %s: %w", poetryLock, err)
	}
	if poetryLockExists {
		return true, fmt.Sprintf("found %s", poetryLock), nil
	}

	pyprojectTomlExists, err := ctx.FileExists(pyprojectToml)
	if err != nil {
		return false, "", fmt.Errorf("checking for %s: %w", pyprojectToml, err)
	}
	if !pyprojectTomlExists {
		return false, fmt.Sprintf("%s not found", pyprojectToml), nil
	}

	pyprojectTomlContent, err := ctx.ReadFile(pyprojectToml)
	if err != nil {
		return false, "", fmt.Errorf("reading %s: %w", pyprojectToml, err)
	}

	var data any
	meta, err := toml.Decode(string(pyprojectTomlContent), &data)
	if err != nil {
		ctx.Warnf("Could not parse %s: %v", pyprojectToml, err)
		return false, fmt.Sprintf("could not parse %s", pyprojectToml), nil
	}

	if meta.IsDefined("tool", "poetry") {
		return true, "found [tool.poetry] in pyproject.toml", nil
	}

	return false, "neither poetry.lock nor [tool.poetry] found", nil
}

// InstallPoetry installs the poetry CLI into a dedicated layer, respecting version constraints.
func InstallPoetry(ctx *gcp.Context) error {
	layer, err := ctx.Layer(poetryLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", poetryLayer, err)
	}

	poetryVersionConstraint, err := RequestedPoetryVersion(ctx)
	if err != nil {
		return err
	}
	installCmd := []string{"python3", "-m", "pip", "install"}
	if poetryVersionConstraint != "" {
		ctx.Logf("Using Poetry version constraint: %s", poetryVersionConstraint)
		installCmd = append(installCmd, fmt.Sprintf("poetry%s", poetryVersionConstraint))
	} else {
		ctx.Logf("No Poetry version constraint found, installing latest.")
		installCmd = append(installCmd, "poetry")
	}

	ctx.Logf("Installing Poetry latest into %s", layer.Path)

	os.Setenv("PYTHONUSERBASE", layer.Path)
	ctx.Logf("Running: %s", strings.Join(installCmd, " "))
	_, err = ctx.Exec(installCmd, gcp.WithUserAttribution)
	os.Unsetenv("PYTHONUSERBASE")
	if err != nil {
		return fmt.Errorf("installing poetry via pip: %w", err)
	}

	ctx.Logf("Poetry installed successfully.")

	binDir := filepath.Join(layer.Path, "bin")
	layer.BuildEnvironment.Prepend("PATH", string(os.PathListSeparator), binDir)

	return nil
}

// RequestedPoetryVersion returns the requested poetry version from pyproject.toml.
func RequestedPoetryVersion(ctx *gcp.Context) (string, error) {
	pyprojectTomlContent, err := ctx.ReadFile(pyprojectToml)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", pyprojectToml, err)
	}

	var parsedTOML struct {
		Tool struct {
			Poetry struct {
				RequiresPoetry string `toml:"requires-poetry"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if _, err := toml.Decode(string(pyprojectTomlContent), &parsedTOML); err != nil {
		return "", fmt.Errorf("could not parse %s to check for poetry version: %w", pyprojectToml, err)
	}

	return parsedTOML.Tool.Poetry.RequiresPoetry, nil
}

// PoetryInstallDependenciesAndConfigureEnv installs dependencies and sets up the runtime environment.
func PoetryInstallDependenciesAndConfigureEnv(ctx *gcp.Context) error {
	layer, err := ctx.Layer(dependenciesLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", dependenciesLayer, err)
	}

	ctx.Logf("Installing application dependencies into %s...", layer.Path)

	execOpts := []gcp.ExecOption{
		gcp.WithUserAttribution,
		gcp.WithEnv(fmt.Sprintf("%s=%s", poetryVenvsPathEnv, layer.Path)),
		gcp.WithEnv("POETRY_VIRTUALENVS_CREATE=true"),
		gcp.WithEnv("POETRY_VIRTUALENVS_IN_PROJECT=false"),
	}

	ctx.Logf("Running: %s", strings.Join(poetryInstallCmd, " "))
	result, err := ctx.Exec(poetryInstallCmd, execOpts...)
	if err != nil {
		return fmt.Errorf("running poetry install: %w", err)
	}
	if result.ExitCode != 0 {
		return gcp.UserErrorf("poetry install failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}
	ctx.Logf("Poetry install successful.")

	// Find the virtual environment path.
	ctx.Logf("Running: %s", strings.Join(poetryEnvInfoCmd, " "))
	pathResult, err := ctx.Exec(poetryEnvInfoCmd, execOpts...)
	if err != nil {
		return fmt.Errorf("getting poetry env info: %w", err)
	}
	if pathResult.ExitCode != 0 {
		return gcp.UserErrorf("poetry env info --path failed with exit code %d: %s", pathResult.ExitCode, pathResult.Stderr)
	}
	venvDir := strings.TrimSpace(pathResult.Stdout)
	if venvDir == "" {
		return fmt.Errorf("could not determine poetry virtual environment path")
	}
	ctx.Logf("Located Poetry virtual environment at: %s", venvDir)

	// Add the venv's bin directory to the PATH.
	layer.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), filepath.Join(venvDir, "bin"))

	return nil
}

// EnsurePoetryLockfile checks for poetry.lock and generates it if it doesn't exist.
func EnsurePoetryLockfile(ctx *gcp.Context) error {
	exists, err := ctx.FileExists(poetryLock)
	if err != nil {
		return err
	}
	if !exists {
		ctx.Warnf("*** To improve build performance, generate and commit %s.", poetryLock)
		ctx.Logf("Running: %s", strings.Join(poetryLockCmd, " "))
		if _, err := ctx.Exec(poetryLockCmd, gcp.WithUserAttribution); err != nil {
			return fmt.Errorf("running poetry lock: %w", err)
		}
	} else {
		ctx.Logf("Using existing %s.", poetryLock)
	}
	return nil
}

// IsUVProject checks if the application is a UV project.
func IsUVProject(ctx *gcp.Context) (bool, string, error) {
	pyprojectTomlExists, err := ctx.FileExists(pyprojectToml)
	if err != nil {
		return false, "", fmt.Errorf("checking for %s: %w", pyprojectToml, err)
	}
	if !pyprojectTomlExists {
		return false, fmt.Sprintf("%s not found", pyprojectToml), nil
	}

	uvLockExists, err := ctx.FileExists(uvLock)
	if err != nil {
		return false, "", fmt.Errorf("checking for %s: %w", uvLock, err)
	}

	if uvLockExists {
		return true, fmt.Sprintf("found %s and %s", pyprojectToml, uvLock), nil
	}

	return true, fmt.Sprintf("found %s", pyprojectToml), nil
}

// RequestedUVVersion returns the requested uv version from pyproject.toml.
func RequestedUVVersion(ctx *gcp.Context) (string, error) {
	pyprojectTomlContent, err := ctx.ReadFile(pyprojectToml)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", pyprojectToml, err)
	}

	var parsedTOML struct {
		Tool struct {
			UV struct {
				RequiredVersion string `toml:"required-version"`
			} `toml:"uv"`
		} `toml:"tool"`
	}
	if _, err := toml.Decode(string(pyprojectTomlContent), &parsedTOML); err != nil {
		return "", fmt.Errorf("could not parse %s to check for uv version: %w", pyprojectToml, err)
	}

	return parsedTOML.Tool.UV.RequiredVersion, nil
}

// InstallUV installs UV into a dedicated layer, respecting version constraints.
func InstallUV(ctx *gcp.Context) error {
	layer, err := ctx.Layer(uvLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", uvLayer, err)
	}

	uvVersionConstraint, err := RequestedUVVersion(ctx)
	if err != nil {
		return fmt.Errorf("getting uv version constraint: %w", err)
	}

	installCmd := []string{"python3", "-m", "pip", "install"}
	if uvVersionConstraint != "" {
		ctx.Logf("Using uv version constraint from pyproject.toml: %s", uvVersionConstraint)
		installCmd = append(installCmd, fmt.Sprintf("uv%s", uvVersionConstraint))
	} else {
		ctx.Logf("No uv version constraint found in pyproject.toml, installing latest.")
		installCmd = append(installCmd, "uv")
	}

	ctx.Logf("Installing uv into %s", layer.Path)

	os.Setenv("PYTHONUSERBASE", layer.Path)
	ctx.Logf("Running: %s", strings.Join(installCmd, " "))
	_, err = ctx.Exec(installCmd)
	os.Unsetenv("PYTHONUSERBASE")
	if err != nil {
		if uvVersionConstraint == "" {
			return buildererror.Errorf(buildererror.StatusInternal, "failed to install uv: %v", err)
		}
		return fmt.Errorf("installing uv with version constraint %s: %w", uvVersionConstraint, err)
	}

	ctx.Logf("uv installed successfully.")

	binDir := filepath.Join(layer.Path, "bin")
	layer.BuildEnvironment.Prepend("PATH", string(os.PathListSeparator), binDir)

	return nil
}

// EnsureUVLockfile checks for uv.lock and generates it if it doesn't exist.
func EnsureUVLockfile(ctx *gcp.Context) error {
	exists, err := ctx.FileExists(uvLock)
	if err != nil {
		return fmt.Errorf("checking for %s: %w", uvLock, err)
	}
	if !exists {
		ctx.Warnf("*** To improve build performance, generate and commit %s.", uvLock)
		ctx.Logf("uv.lock not found, generating it using `uv lock`...")
		if _, err := ctx.Exec(uvLockCmd, gcp.WithUserAttribution); err != nil {
			return fmt.Errorf("failed to generate uv.lock with uv: %w", err)
		}
		ctx.Logf("uv.lock generated successfully.")
	} else {
		ctx.Logf("Using existing %s.", uvLock)
	}
	return nil
}

// UVInstallDependenciesAndConfigureEnv installs dependencies and sets up the runtime environment using uv.
func UVInstallDependenciesAndConfigureEnv(ctx *gcp.Context) error {
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

	ctx.Logf("Installing dependencies with `uv sync` into the virtual environment...")
	if _, err := ctx.Exec(uvSyncCmd, gcp.WithUserAttribution, gcp.WithEnv("VIRTUAL_ENV="+venvDir)); err != nil {
		return fmt.Errorf("failed to sync dependencies with uv: %w", err)
	}
	ctx.Logf("Dependencies installed to virtual environment at %s", venvDir)

	venvBinDir := filepath.Join(venvDir, "bin")
	layer.SharedEnvironment.Prepend("PATH", string(filepath.ListSeparator), venvBinDir)
	return nil
}
