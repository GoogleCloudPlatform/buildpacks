// Copyright 2020 Google LLC
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

// Package python contains Python buildpack library code.
package python

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

const (
	dateFormat = time.RFC3339Nano
	// expirationTime is an arbitrary amount of time of 1 day to refresh the cache layer.
	expirationTime = time.Duration(time.Hour * 24)

	pythonVersionKey   = "python_version"
	dependencyHashKey  = "dependency_hash"
	expiryTimestampKey = "expiry_timestamp"

	cacheName = "pipcache"

	// RequirementsFilesEnv is an environment variable containg os-path-separator-separated list of paths to pip requirements files.
	// The requirements files are processed from left to right, with requirements from the next overriding any conflicts from the previous.
	RequirementsFilesEnv = "GOOGLE_INTERNAL_REQUIREMENTS_FILES"

	// VendorPipDepsEnv is the envar used to opt using vendored pip dependencies
	VendorPipDepsEnv = "GOOGLE_VENDOR_PIP_DEPENDENCIES"

	// DefaultPipTargetDir is the default directory to install python dependencies.
	DefaultPipTargetDir = "lib"

	versionFile = ".python-version"
	versionKey  = "version"
	versionEnv  = "GOOGLE_PYTHON_VERSION"

	// python37SharedLibDir is the location of the shared Python library when building the python37 runtime.
	python37SharedLibDir = "/layers/google.python.runtime/python/lib/python3.7/config-3.7m-x86_64-linux-gnu"
	// python38SharedLibDir is the location of the shared Python library when building the python38 runtime.
	python38SharedLibDir = "/layers/google.python.runtime/python/lib/python3.8/config-3.8-x86_64-linux-gnu"
)

var (
	// RequirementsProvides denotes that the buildpack provides requirements.txt in the environment.
	RequirementsProvides = []libcnb.BuildPlanProvide{{Name: "requirements.txt"}}
	// RequirementsRequires denotes that the buildpack consumes requirements.txt from the environment.
	RequirementsRequires = []libcnb.BuildPlanRequire{{Name: "requirements.txt"}}
	// RequirementsProvidesPlan is a build plan returned by buildpacks that provide requirements.txt.
	RequirementsProvidesPlan = libcnb.BuildPlan{Provides: RequirementsProvides}
	// RequirementsProvidesRequiresPlan is a build plan returned by buildpacks that consume requirements.txt.
	RequirementsProvidesRequiresPlan = libcnb.BuildPlan{Provides: RequirementsProvides, Requires: RequirementsRequires}

	// latestPythonVersionPerStack is the latest Python version per stack to use if not specified by the user.
	latestPythonVersionPerStack = map[string]string{
		runtime.Ubuntu1804: "3.9.*",
		runtime.Ubuntu2204: "3.13.*",
		runtime.Ubuntu2404: "3.14.*",
	}

	execPrefixRegex = regexp.MustCompile(`exec_prefix\s*=\s*"([^"]+)`)
)

// SysconfigPatcherCapability is the capability key for the SysconfigPatcher.
const SysconfigPatcherCapability = "python.SysconfigPatcher"

// sysconfigPatcher is an interface for patching python sysconfig.
type sysconfigPatcher interface {
	PatchSysconfig(ctx *gcp.Context, layer *libcnb.Layer) error
}

// PipInstallerCapability is the capability key for the PipInstaller.
const PipInstallerCapability = "python.PipInstaller"

// pipInstaller is an interface for installing python dependencies.
type pipInstaller interface {
	Install(ctx *gcp.Context, l *libcnb.Layer, reqs ...string) error
}

// Version returns the installed version of Python.
func Version(ctx *gcp.Context) (string, error) {
	result, err := ctx.Exec([]string{"python3", "--version"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// RuntimeVersion validate and returns the customer requested Python version by inspecting the
// environment variables and .python-version file.
func RuntimeVersion(ctx *gcp.Context, dir string) (string, error) {
	if v := os.Getenv(env.Runtime); v != "" && !strings.HasPrefix(v, "python") {
		return "*", nil
	}

	if v := os.Getenv(versionEnv); v != "" {
		ctx.Logf("Using Python version from %s: %s", versionEnv, v)
		return v, nil
	}
	if v := os.Getenv(env.RuntimeVersion); v != "" {
		ctx.Logf("Using Python version from %s: %s", env.RuntimeVersion, v)
		return v, nil
	}
	v, err := versionFromFile(ctx, dir)
	if err != nil {
		return "", err
	}
	if v != "" {
		return v, nil
	}

	os := runtime.OSForStack(ctx)

	latestPythonVersionForStack, ok := latestPythonVersionPerStack[os]
	if !ok {
		return "", gcp.UserErrorf("invalid stack for Python runtime: %q", os)
	}

	ctx.Logf("Python version not specified, using the latest available Python runtime for the stack %q", os)
	return latestPythonVersionForStack, nil
}

// SupportsSmartDefaultEntrypoint returns true if the runtime version supports smart default entrypoint.
func SupportsSmartDefaultEntrypoint(ctx *gcp.Context) (bool, error) {
	v, err := RuntimeVersion(ctx, ctx.ApplicationRoot())
	if err != nil {
		return false, err
	}
	// If the version contains a wildcard, we will replace with 0 for the semver comparison.
	v = strings.ReplaceAll(v, "*", "0")

	return versionMatchesSemver(ctx, ">=3.13.0-0", v)
}

// versionMatchesSemver checks if the provided version matches the given version semver range.
// The range string has the following format: https://github.com/blang/semver#ranges.
func versionMatchesSemver(ctx *gcp.Context, versionRange string, version string) (bool, error) {
	if version == "" {
		return false, nil
	}
	if isSupportedUnstablePythonVersion(version) {
		// The format of Python pre-release version e.g. 3.14.0rc1 doesn't follow the semver rule
		// that requires a hyphen before the identifier "rc".
		if strings.Contains(version, "rc") && !strings.Contains(version, "-rc") {
			version = strings.Replace(version, "rc", "-rc", 1)
		}
	}
	constraint, err := semver.NewConstraint(versionRange)
	if err != nil {
		return false, fmt.Errorf("invalid version range %q: %w", versionRange, err)
	}
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", version, err)
	}
	if !constraint.Check(v) {
		ctx.Debugf("Python version %q does not match the semver constraint %q", version, versionRange)
		return false, nil
	}
	return true, nil
}

func versionFromFile(ctx *gcp.Context, dir string) (string, error) {
	vf := filepath.Join(dir, versionFile)
	versionFileExists, err := ctx.FileExists(vf)
	if err != nil {
		return "", err
	}
	if versionFileExists {
		raw, err := ctx.ReadFile(vf)
		if err != nil {
			return "", err
		}
		v := strings.TrimSpace(string(raw))
		if v != "" {
			ctx.Logf("Using Python version from %s: %s", vf, v)
			return v, nil
		}
		return "", gcp.UserErrorf("%s exists but does not specify a version", vf)
	}
	return "", nil
}

// PIPInstallRequirements installs dependencies from the given requirements files in a virtual env.
// It will install the files in order in which they are specified, so that dependencies specified
// in later requirements files can override later ones.
//
// This function is responsible for installing requirements files for all buildpacks that require
// it. The buildpacks used to install requirements into separate layers and add the layer path to
// PYTHONPATH. However, this caused issues with some packages as it would allow users to
// accidentally override some builtin stdlib modules, e.g. typing, enum, etc., and cause both
// build-time and run-time failures.
func PIPInstallRequirements(ctx *gcp.Context, l *libcnb.Layer, reqs ...string) error {
	if cap := ctx.Capability(PipInstallerCapability); cap != nil {
		i, ok := cap.(pipInstaller)
		if !ok {
			return gcp.InternalErrorf("capability %q does not implement pipInstaller interface", PipInstallerCapability)
		}
		return i.Install(ctx, l, reqs...)
	}

	shouldInstall, err := prepareDependenciesLayer(ctx, l, "pip", reqs...)
	if err != nil {
		return err
	}
	if !shouldInstall {
		ctx.Logf("Application dependencies are up to date, skipping installation.")
		return nil
	}

	ctx.Logf("Installing application dependencies.")
	if err := ar.GeneratePythonConfig(ctx); err != nil {
		return fmt.Errorf("generating Artifact Registry credentials: %w", err)
	}

	// History of the logic below:
	//
	// pip install --target has several subtle issues:
	// We cannot use --upgrade: https://github.com/pypa/pip/issues/8799.
	// We also cannot _not_ use --upgrade, see the requirements_bin_conflict acceptance test.
	//
	// Instead, we use Python per-user site-packages (https://www.python.org/dev/peps/pep-0370/)
	// where we can and virtualenv where we cannot.
	//
	// Each requirements file is installed separately to allow the requirements.txt files
	// to specify conflicting dependencies (e.g. functions-framework pins package A at 1.2.0 but
	// the user's requirements.txt file pins A at 1.4.0. The user should be able to override
	// the functions-framework-pinned package).

	// HACK: For backwards compatibility with Python 3.7 and 3.8 on App Engine and Cloud Functions.
	virtualEnv := requiresVirtualEnv()
	if virtualEnv {
		// --without-pip and --system-site-packages allow us to use `pip` and other packages from the
		// build image and avoid reinstalling them, saving about 10MB.
		// TODO(b/140775593): Use virtualenv pip after FTL is no longer used and remove from build image.
		if _, err := ctx.Exec([]string{"python3", "-m", "venv", "--without-pip", "--system-site-packages", l.Path}); err != nil {
			return err
		}
		if err := copySharedLibs(ctx, l); err != nil {
			return err
		}

		// The VIRTUAL_ENV variable is usually set by the virtual environment's activate script.
		l.SharedEnvironment.Override("VIRTUAL_ENV", l.Path)
		// Use the virtual environment python3 for all subsequent commands in this buildpack, for
		// subsequent buildpacks, l.Path/bin will be added by lifecycle.
		if err := ctx.Setenv("PATH", filepath.Join(l.Path, "bin")+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
			return err
		}
		if err := ctx.Setenv("VIRTUAL_ENV", l.Path); err != nil {
			return err
		}
	} else {
		l.SharedEnvironment.Default("PYTHONUSERBASE", l.Path)
		if err := ctx.Setenv("PYTHONUSERBASE", l.Path); err != nil {
			return err
		}
	}

	for _, req := range reqs {
		cmd := basePipInstallArgs(req)
		cmd = append(cmd,
			"--no-cache-dir", // We used to save this to a layer, but it made builds slower because it includes http caching of pypi requests.
		)
		cmd = appendVendoringFlags(cmd)
		if !virtualEnv {
			cmd = append(cmd, "--user") // Install into user site-packages directory.
		}
		if _, err := ctx.Exec(cmd,
			gcp.WithUserAttribution); err != nil {
			return err
		}
	}

	if err := compileBytecode(ctx, l.Path); err != nil {
		return err
	}

	return CheckIncompatibleDependencies(ctx)
}

// CheckIncompatibleDependencies checks for incompatible dependencies using pip check.
func CheckIncompatibleDependencies(ctx *gcp.Context) error {
	ctx.Logf("Checking for incompatible dependencies.")
	result, err := ctx.Exec([]string{"python3", "-m", "pip", "check"}, gcp.WithUserAttribution)
	if result == nil {
		return fmt.Errorf("pip check: %w", err)
	}
	if result.ExitCode == 0 {
		return nil
	}
	pyVer, err := Version(ctx)
	if err != nil {
		return err
	}
	// HACK: For backwards compatibility on App Engine and Cloud Functions Python 3.7 only report a
	//   warning.
	if strings.HasPrefix(pyVer, "Python 3.7") {
		ctx.Warnf("Found incompatible dependencies: %q", result.Stdout)
		return nil
	}
	return gcp.UserErrorf("found incompatible dependencies: %q", result.Stdout)
}

// MakerPipInstaller implements the PipInstaller interface for the maker tool.
type MakerPipInstaller struct{}

// Install installs python dependencies to the target directory specified by GOOGLE_PIP_TARGET_DIR.
// If GOOGLE_PIP_TARGET_DIR is not set, it defaults to "lib".
func (i MakerPipInstaller) Install(ctx *gcp.Context, l *libcnb.Layer, reqs ...string) error {
	targetDir := os.Getenv(env.PipTargetDir)
	if targetDir == "" {
		targetDir = DefaultPipTargetDir
	}

	ctx.Logf("Installing dependencies to target directory: %s", targetDir)
	var targetPath string
	if filepath.IsAbs(targetDir) {
		targetPath = targetDir
	} else {
		targetPath = filepath.Join(ctx.ApplicationRoot(), targetDir)
	}

	// We set PYTHONPATH so that the python interpreter can find the dependencies in the target
	// directory. We use Override so that it takes precedence over any existing PYTHONPATH.
	l.LaunchEnvironment.Override("PYTHONPATH", targetDir)
	l.SharedEnvironment.Override("PYTHONPATH", targetPath)

	ctx.AddLabel("ENV_PYTHONPATH", targetDir)

	for _, req := range reqs {
		cmd := basePipInstallArgs(req)
		cmd = appendVendoringFlags(cmd)
		cmd = append(cmd, "--target", targetPath)

		if _, err := ctx.Exec(cmd, gcp.WithUserAttribution); err != nil {
			return err
		}
	}

	return nil
}

// prepareDependenciesLayer is a helper function that handles the common logic for preparing a dependency layer.
// It checks for empty requirements, manages the cache, and clears the layer if necessary.
// It returns a boolean indicating whether the installation should proceed.
func prepareDependenciesLayer(ctx *gcp.Context, l *libcnb.Layer, installerName string, reqs ...string) (bool, error) {
	// Defensive check
	if len(reqs) == 0 {
		ctx.Debugf("No requirements files to install, clearing layer.")
		if err := ctx.ClearLayer(l); err != nil {
			return false, fmt.Errorf("clearing layer %q: %w", l.Name, err)
		}
		return false, nil
	}

	// Caching logic
	currentPythonVersion, err := Version(ctx)
	if err != nil {
		return false, err
	}
	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey,
		cache.WithFiles(reqs...),
		cache.WithStrings(currentPythonVersion, installerName))
	if err != nil {
		return false, err
	}

	// Check cache expiration to pick up new versions of dependencies that are not pinned.
	expired := cacheExpired(ctx, l)

	if cached && !expired {
		ctx.CacheHit(l.Name)
		return false, nil
	}
	ctx.CacheMiss(l.Name)

	if expired {
		ctx.Debugf("Dependencies cache expired, clearing layer.")
	}
	if err := ctx.ClearLayer(l); err != nil {
		return false, fmt.Errorf("clearing layer %q: %w", l.Name, err)
	}

	// Update layer metadata for caching
	cache.Add(ctx, l, dependencyHashKey, hash)
	ctx.SetMetadata(l, pythonVersionKey, currentPythonVersion)
	ctx.SetMetadata(l, expiryTimestampKey, time.Now().Add(expirationTime).Format(dateFormat))

	return true, nil
}

// compileBytecode is a helper that generates deterministic hash-based pyc files for faster startup.
func compileBytecode(ctx *gcp.Context, path string) error {
	// Generate deterministic hash-based pycs (https://www.python.org/dev/peps/pep-0552/).
	// Use the unchecked version to skip hash validation at run time (for faster startup).
	result, err := ctx.Exec([]string{
		"python3", "-m", "compileall",
		"--invalidation-mode", "unchecked-hash",
		"-qq", // Do not print any message (matches `pip install` behavior).
		path,
	}, gcp.WithUserAttribution)

	if err != nil {
		if result != nil {
			if result.ExitCode == 1 {
				// Ignore file compilation errors (matches `pip install` behavior).
				return nil
			}
			return fmt.Errorf("compileall: %s", result.Combined)
		}
		return fmt.Errorf("compileall: %v", err)
	}
	return nil
}

// cacheExpired returns true when the cache is past expiration.
func cacheExpired(ctx *gcp.Context, l *libcnb.Layer) bool {
	t := time.Now()
	expiry := ctx.GetMetadata(l, expiryTimestampKey)
	if expiry != "" {
		var err error
		t, err = time.Parse(dateFormat, expiry)
		if err != nil {
			ctx.Debugf("Could not parse expiration date %q, assuming now: %v", expiry, err)
		}
	}
	return !t.After(time.Now())
}

// requiresVirtualEnv returns true for runtimes that require a virtual environment to be created before pip install.
// We cannot use Python per-user site-packages (https://www.python.org/dev/peps/pep-0370/),
// because Python 3.7 and 3.8 on App Engine and Cloud Functions have a virtualenv set up
// that disables user site-packages. The base images include a virtual environment pointing to
// a directory that is not writeable in the buildpacks world (/env). In order to keep
// compatiblity with base image updates, we replace the virtual environment with a writeable one.
func requiresVirtualEnv() bool {
	runtime := os.Getenv(env.Runtime)
	return runtime == "python37" || runtime == "python38"
}

// copySharedLibs moves the shared libs from the runtime layer into pip layer. This is required to
// support building native extensions in python37 and python38 because virtual env does not copy
// the correctly.
func copySharedLibs(ctx *gcp.Context, l *libcnb.Layer) error {
	var oldPath string
	var newPath string
	if os.Getenv(env.Runtime) == "python37" {
		oldPath = python37SharedLibDir
		newPath = filepath.Join(l.Path, "lib", "python3.7", filepath.Base(oldPath))
	}
	if os.Getenv(env.Runtime) == "python38" {
		oldPath = python38SharedLibDir
		newPath = filepath.Join(l.Path, "lib", "python3.8", filepath.Base(oldPath))
	}
	exists, err := ctx.FileExists(oldPath)
	if err != nil {
		return gcp.InternalErrorf("finding shared libs in %v: %w", oldPath, err)
	}
	if exists {
		if err := os.Symlink(oldPath, newPath); err != nil {
			return gcp.InternalErrorf("symlinking shared libs from %v to %v: %w", oldPath, newPath, err)
		}
	}
	return nil
}

// Checks if the Python version is an unstable supported release candidate.
func isSupportedUnstablePythonVersion(constraint string) bool {
	return strings.Count(constraint, ".") == 2 && strings.Count(constraint, "rc") == 1
}

// isPackageManagerConfigured checks if the environment is configured to use the specified package manager.
func isPackageManagerConfigured(pm string) bool {
	pmPreference := os.Getenv(env.PythonPackageManager)
	return strings.EqualFold(pmPreference, pm) // Case insensitive comparison.
}

// appendVendoringFlags checks for and appends vendored dependency flags to the command.
func appendVendoringFlags(cmd []string) []string {
	if vendorDir, isVendored := os.LookupEnv(VendorPipDepsEnv); isVendored {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PipVendorDependenciesCounterID).Increment(1)
		return append(cmd, "--no-index", "--find-links", vendorDir)
	}
	return cmd
}

func isPackageManagerEmpty() bool {
	return os.Getenv(env.PythonPackageManager) == ""
}

func isUVDefaultPackageManagerForRequirements(ctx *gcp.Context) bool {
	v, err := RuntimeVersion(ctx, ctx.ApplicationRoot())
	if err != nil {
		return false
	}
	v = strings.ReplaceAll(v, "*", "0")
	isPythonVersionGreaterThanEqualTo314, err := versionMatchesSemver(ctx, ">=3.14.0-0", v)
	if err != nil {
		return false
	}
	return isPythonVersionGreaterThanEqualTo314
}

// isBothPyprojectAndRequirementsPresent checks if both pyproject.toml and requirements.txt are present.
func isBothPyprojectAndRequirementsPresent(ctx *gcp.Context) bool {
	pyprojectTomlExists, _ := ctx.FileExists(pyprojectToml)
	requirementsTxtExists, _ := ctx.FileExists(requirements)
	return pyprojectTomlExists && requirementsTxtExists
}

// PatchSysconfig patches the python sysconfig variable prefix.
func PatchSysconfig(ctx *gcp.Context, layer *libcnb.Layer) error {
	if cap := ctx.Capability(SysconfigPatcherCapability); cap != nil {
		p, ok := cap.(sysconfigPatcher)
		if !ok {
			return gcp.InternalErrorf("capability %q must implement sysconfigPatcher", SysconfigPatcherCapability)
		}
		return p.PatchSysconfig(ctx, layer)
	}
	// replace python sysconfig variable prefix from "/opt/python" to "/layers/google.python.runtime/python/" which is the layer.Path
	// python is installed in /layers/google.python.runtime/python/ for unified builder,
	// while the python downloaded from debs is installed in "/opt/python".
	sysconfig, err := ctx.Exec([]string{filepath.Join(layer.Path, "bin/python3"), "-m", "sysconfig"})
	if err != nil {
		ctx.Warnf("Getting python sysconfig: %v", err)
	}
	execPrefix, err := parseExecPrefix(sysconfig.Stdout)
	if err != nil {
		return err
	}
	result, err := ctx.Exec([]string{
		"grep",
		"-rlI",
		execPrefix,
		layer.Path,
	})
	if err != nil {
		ctx.Warnf("Grep failed: %v", err)
	}
	paths := strings.Split(result.Stdout, "\n")
	for _, path := range paths {
		if _, err := ctx.Exec([]string{
			"sed",
			"-i",
			"s|" + execPrefix + "|" + layer.Path + "|g",
			path,
		}); err != nil {
			ctx.Warnf("Patching file %q: %v", path, err)
		}
	}
	return nil
}

func parseExecPrefix(sysconfig string) (string, error) {
	match := execPrefixRegex.FindStringSubmatch(sysconfig)
	if len(match) < 2 {
		return "", fmt.Errorf("determining Python exec prefix: %v", match)
	}
	return match[1], nil
}

// MakerSysconfigPatcher implements the sysconfigPatcher interface for the maker tool.
type MakerSysconfigPatcher struct{}

// PatchSysconfig does nothing, as no patching is required for maker.
func (p MakerSysconfigPatcher) PatchSysconfig(ctx *gcp.Context, layer *libcnb.Layer) error {
	return nil
}

func basePipInstallArgs(req string) []string {
	return []string{
		"python3", "-m", "pip", "install",
		"--requirement", req,
		"--upgrade",
		"--upgrade-strategy", "only-if-needed",
		"--no-warn-script-location",   // bin is added at run time by lifecycle.
		"--no-warn-conflicts",         // Needed for python37 and maker mode which allowed users to override dependencies. For newer versions, we do a separate `pip check`.
		"--force-reinstall",           // Some dependencies may be in the build image but not run image. Later requirements.txt should override earlier.
		"--no-compile",                // Prevent default timestamp-based bytecode compilation. Deterministic pycs are generated in a second step below.
		"--disable-pip-version-check", // If we were going to upgrade pip, we would have done it already in the runtime buildpack.
	}
}
