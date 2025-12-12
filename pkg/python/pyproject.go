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
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb/v2"
)

const (
	pip                = "pip"
	poetry             = "poetry"
	poetryLayer        = "poetry"
	dependenciesLayer  = "poetry-dependencies"
	pyprojectToml      = "pyproject.toml"
	poetryLock         = "poetry.lock"
	poetryVenvsPathEnv = "POETRY_VIRTUALENVS_PATH"
	uv                 = "uv"
	uvLock             = "uv.lock"
	uvLayer            = "uv"
)

var (
	poetryInstallCmd = []string{"poetry", "install", "--no-interaction", "--sync", "--only", "main", "--no-root"}
	poetryEnvInfoCmd = []string{"poetry", "env", "info", "--path"}
	poetryLockCmd    = []string{"poetry", "lock", "--no-interaction"}
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

// InstallPoetry installs the poetry CLI using uv to speed up the installation.
func InstallPoetry(ctx *gcp.Context) error {
	err := installTool(ctx, pip, uv, uvLayer, "")
	if err != nil {
		return fmt.Errorf("installing uv: %w", err)
	}

	poetryVersionConstraint, err := RequestedPoetryVersion(ctx)
	if err != nil {
		return fmt.Errorf("getting poetry version constraint: %w", err)
	}

	if err := installTool(ctx, uv, poetry, poetryLayer, poetryVersionConstraint); err != nil {
		return fmt.Errorf("installing poetry with uv: %w", err)
	}

	ctx.Logf("Uninstalling uv to remove it from the final image.")
	uninstallUVCmd := []string{"python3", "-m", "pip", "uninstall", "-y", uv}
	if _, err := ctx.Exec(uninstallUVCmd); err != nil {
		ctx.Warnf("Failed to uninstall uv, it may remain in the final image: %v", err)
	}

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
	return ensureLockfile(ctx, "poetry", poetryLock, poetryLockCmd)
}

// IsUVPyproject checks if the application is a UV Pyproject application.
func IsUVPyproject(ctx *gcp.Context) (bool, string, error) {
	if isBothPyprojectAndRequirementsPresent(ctx) {
		return false, fmt.Sprintf("%s and %s found, prefer requirements.txt", pyprojectToml, requirements), nil
	}

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

	// When no lockfile is present, check the environment variable to decide the package manager.
	if isPackageManagerConfigured(uv) {
		return true, fmt.Sprintf("found %s, using uv because %s is set to 'uv'", pyprojectToml, env.PythonPackageManager), nil
	}
	if isPackageManagerEmpty() {
		return true, fmt.Sprintf("found %s and %s is not set, using uv as default package manager", pyprojectToml, env.PythonPackageManager), nil
	}
	return false, fmt.Sprintf("found %s, but %s is not set to 'uv'", pyprojectToml, env.PythonPackageManager), nil
}

// RequestedUVVersion returns the requested uv version from pyproject.toml.
func RequestedUVVersion(ctx *gcp.Context) (string, error) {
	pyprojectTomlexists, err := ctx.FileExists(pyprojectToml)
	if err != nil {
		return "", fmt.Errorf("checking for %s: %w", pyprojectToml, err)
	}
	if !pyprojectTomlexists {
		return "", nil
	}

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

// IsUVInstalledInPath checks if the uv executable is installed in the PATH.
func IsUVInstalledInPath(ctx *gcp.Context) (bool, error) {
	result, err := ctx.Exec([]string{"uv", "--version"})
	if err != nil {
		ctx.Debugf("uv --version failed with error: %v", err)
		return false, nil
	}
	if result.ExitCode != 0 {
		ctx.Debugf("uv --version exited with code %d", result.ExitCode)
		return false, nil
	}
	return true, nil
}

// InstallUV installs UV into a dedicated layer, respecting version constraints.
func InstallUV(ctx *gcp.Context) error {
	uvVersionConstraint, err := RequestedUVVersion(ctx)
	if err != nil {
		return fmt.Errorf("getting uv version constraint: %w", err)
	}
	if uvVersionConstraint == "" {
		isInstalled, err := IsUVInstalledInPath(ctx)
		if err != nil {
			return fmt.Errorf("checking for pre-installed uv: %w", err)
		}
		if isInstalled {
			return nil
		}
	}
	return installTool(ctx, pip, uv, uvLayer, uvVersionConstraint)
}

// EnsureUVLockfile checks for uv.lock and generates it if it doesn't exist.
func EnsureUVLockfile(ctx *gcp.Context) error {
	lockCmd := []string{"uv", "lock"}
	lockCmd = appendVendoringFlags(lockCmd)
	return ensureLockfile(ctx, "uv", uvLock, lockCmd)
}

// UVInstallDependenciesAndConfigureEnv installs dependencies and sets up the runtime environment using uv.
func UVInstallDependenciesAndConfigureEnv(ctx *gcp.Context, l *libcnb.Layer) (string, error) {
	pythonVersion, err := Version(ctx)
	if err != nil {
		return "", err
	}
	pythonVersion = strings.TrimPrefix(pythonVersion, "Python ")

	venvDir := filepath.Join(l.Path, ".venv")
	ctx.Logf("Creating virtual environment at %s with Python %s", venvDir, pythonVersion)
	venvCmd := []string{"uv", "venv", venvDir, "--python", pythonVersion}
	if _, err := ctx.Exec(venvCmd, gcp.WithUserAttribution); err != nil {
		return "", fmt.Errorf("failed to create virtual environment with uv: %w", err)
	}

	syncCmd := []string{"uv", "sync", "--active", "--link-mode=copy", "--python", pythonVersion}
	syncCmd = appendVendoringFlags(syncCmd)

	ctx.Logf("Installing dependencies with `uv sync` into the virtual environment...")
	if _, err := ctx.Exec(syncCmd, gcp.WithUserAttribution, gcp.WithEnv("VIRTUAL_ENV="+venvDir)); err != nil {
		return "", fmt.Errorf("failed to sync dependencies with uv: %w", err)
	}
	ctx.Logf("Dependencies installed to virtual environment at %s", venvDir)

	venvBinDir := filepath.Join(venvDir, "bin")
	l.SharedEnvironment.Prepend("PATH", string(filepath.ListSeparator), venvBinDir)
	return venvDir, nil
}

// installTool handles the common logic of installing a python tool with either pip or uv.
func installTool(ctx *gcp.Context, provider, toolName, layerName, versionConstraint string) error {
	layer, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	dependencyToInstall := toolName
	if versionConstraint != "" {
		ctx.Logf("Using %s version constraint: %s", toolName, versionConstraint)
		dependencyToInstall = fmt.Sprintf("%s%s", toolName, versionConstraint)
	} else {
		ctx.Logf("No %s version constraint found, installing latest", toolName)
	}

	var installCmd []string
	switch provider {
	case "pip":
		installCmd = []string{"python3", "-m", "pip", "install", dependencyToInstall}
		ctx.Logf("Installing %s into %s using pip", dependencyToInstall, layer.Path)
	case "uv":
		installCmd = []string{"uv", "pip", "install", "--system", "--link-mode=copy", "-q", dependencyToInstall}
		ctx.Logf("Installing %s into %s using uv", dependencyToInstall, layer.Path)
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	os.Setenv("PYTHONUSERBASE", layer.Path)
	ctx.Logf("Running: %s", strings.Join(installCmd, " "))
	_, err = ctx.Exec(installCmd, gcp.WithUserAttribution)
	os.Unsetenv("PYTHONUSERBASE")
	if err != nil {
		if versionConstraint == "" {
			return buildererror.Errorf(buildererror.StatusInternal, "failed to install %s with %s: %v", toolName, provider, err)
		}
		return fmt.Errorf("installing %s with version constraint %s: %w", toolName, versionConstraint, err)
	}

	ctx.Logf("%s installed successfully.", toolName)

	binDir := filepath.Join(layer.Path, "bin")
	layer.BuildEnvironment.Prepend("PATH", string(os.PathListSeparator), binDir)

	return nil
}

// ensureLockfile handles the common logic of checking/generating a lockfile for a given tool.
func ensureLockfile(ctx *gcp.Context, toolName, lockFile string, lockCmd []string) error {
	exists, err := ctx.FileExists(lockFile)
	if err != nil {
		return fmt.Errorf("checking for %s: %w", lockFile, err)
	}
	if exists {
		ctx.Logf("Using existing %s.", lockFile)
		return nil
	}
	ctx.Logf("%s not found, generating it using `%s`...", lockFile, strings.Join(lockCmd, " "))
	if _, err := ctx.Exec(lockCmd, gcp.WithUserAttribution); err != nil {
		return fmt.Errorf("failed to generate %s with %s: %w", lockFile, toolName, err)
	}
	ctx.Logf("%s generated successfully.", lockFile)
	return nil
}

// GetScriptCommand returns the script command from pyproject.toml if it exists.
func GetScriptCommand(ctx *gcp.Context) ([]string, error) {
	pyprojectTomlContent, err := ctx.ReadFile(pyprojectToml)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pyprojectToml, err)
	}

	var parsedTOML struct {
		Project struct {
			Scripts map[string]string `toml:"scripts"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Scripts map[string]string `toml:"scripts"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	if _, err := toml.Decode(string(pyprojectTomlContent), &parsedTOML); err != nil {
		ctx.Warnf("Could not parse %s: %v", pyprojectToml, err)
		return nil, nil
	}

	if len(parsedTOML.Tool.Poetry.Scripts) > 0 {
		if len(parsedTOML.Tool.Poetry.Scripts) == 1 {
			for scriptName := range parsedTOML.Tool.Poetry.Scripts {
				return []string{scriptName}, nil
			}
		}
		if parsedTOML.Tool.Poetry.Scripts["start"] != "" {
			return []string{"start"}, nil
		}
	}

	if len(parsedTOML.Project.Scripts) > 0 {
		if len(parsedTOML.Project.Scripts) == 1 {
			for scriptName := range parsedTOML.Project.Scripts {
				return []string{scriptName}, nil
			}
		}
		if parsedTOML.Project.Scripts["start"] != "" {
			return []string{"start"}, nil
		}
	}

	return nil, nil
}

// IsPyprojectEnabled checks if the pyproject feature is enabled.
// For any future changes to the release stage, this is the single place to make changes.
func IsPyprojectEnabled(ctx *gcp.Context) bool {
	pyprojectTomlExists, _ := ctx.FileExists(pyprojectToml)
	if !pyprojectTomlExists {
		return false
	}
	if isBothPyprojectAndRequirementsPresent(ctx) {
		return false // Prefer requirements.txt over pyproject.toml.
	}
	// TODO(b/468177624): Remove this check to move pyproject to GA for all Python versions.
	v, err := RuntimeVersion(ctx, ctx.ApplicationRoot())
	if err != nil {
		return false
	}
	v = strings.ReplaceAll(v, "*", "0")
	isPythonVersionGreaterThanEqualTo313, err := versionMatchesSemver(ctx, ">=3.13.0-0", v)
	if err != nil {
		return false
	}
	return isPythonVersionGreaterThanEqualTo313 || env.IsBetaSupported()
}

// IsPipPyproject checks if the application is a pip pyproject.
func IsPipPyproject(ctx *gcp.Context) bool {
	if isBothPyprojectAndRequirementsPresent(ctx) {
		return false // Prefer requirements.txt over pyproject.toml.
	}
	return isPackageManagerConfigured(pip) && IsPyprojectEnabled(ctx) && (env.IsGCP() || env.IsGCF())
}

// PipInstallPyproject installs dependencies from a pyproject.toml file in the current directory.
func PipInstallPyproject(ctx *gcp.Context, l *libcnb.Layer) error {
	ctx.Logf("Installing application dependencies from pyproject.toml.")
	currentPythonVersion, err := Version(ctx)
	if err != nil {
		return err
	}
	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey,
		cache.WithFiles("pyproject.toml"),
		cache.WithStrings(currentPythonVersion))
	if err != nil {
		return err
	}

	// Check cache expiration to pick up new versions of dependencies that are not pinned.
	expired := cacheExpired(ctx, l)

	if cached && !expired {
		ctx.Logf("Dependencies cached and not expired. Skipping installation.")
		return nil
	}

	if expired {
		ctx.Debugf("Dependencies cache expired, clearing layer.")
	}

	if err := ctx.ClearLayer(l); err != nil {
		return fmt.Errorf("clearing layer %q: %w", l.Name, err)
	}

	cache.Add(ctx, l, dependencyHashKey, hash)
	// Update the layer metadata.
	ctx.SetMetadata(l, pythonVersionKey, currentPythonVersion)
	ctx.SetMetadata(l, expiryTimestampKey, time.Now().Add(expirationTime).Format(dateFormat))

	if err := ar.GeneratePythonConfig(ctx); err != nil {
		return fmt.Errorf("generating Artifact Registry credentials: %w", err)
	}

	cmd := []string{
		"python3", "-m", "pip", "install",
		".",
		"--upgrade",
		"--upgrade-strategy", "only-if-needed",
		"--no-warn-script-location",   // bin is added at run time by lifecycle.
		"--disable-pip-version-check", // If we were going to upgrade pip, we would have done it already in the runtime buildpack.
		"--no-cache-dir",              // We don't want http caching of pypi requests.
	}
	vendorDir, isVendored := os.LookupEnv(VendorPipDepsEnv)
	if isVendored {
		cmd = append(cmd, "--no-index", "--find-links", vendorDir)
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PipVendorDependenciesCounterID).Increment(1)
	}
	if _, err := ctx.Exec(cmd,
		gcp.WithUserAttribution); err != nil {
		return err
	}

	return nil
}
