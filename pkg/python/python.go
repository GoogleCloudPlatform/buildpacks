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
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
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
)

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

	// This will use the highest listed at https://dl.google.com/runtimes/python/version.json.
	ctx.Logf("Python version not specified, using the latest available version.")
	return "*", nil
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

// InstallRequirements installs dependencies from the given requirements files in a virtual env.
// It will install the files in order in which they are specified, so that dependencies specified
// in later requirements files can override later ones.
//
// This function is responsible for installing requirements files for all buildpacks that require
// it. The buildpacks used to install requirements into separate layers and add the layer path to
// PYTHONPATH. However, this caused issues with some packages as it would allow users to
// accidentally override some builtin stdlib modules, e.g. typing, enum, etc., and cause both
// build-time and run-time failures.
func InstallRequirements(ctx *gcp.Context, l *libcnb.Layer, reqs ...string) error {
	// Defensive check, this should not happen in practice.
	if len(reqs) == 0 {
		ctx.Debugf("No requirements.txt to install, clearing layer.")
		if err := ctx.ClearLayer(l); err != nil {
			return fmt.Errorf("clearing layer %q: %w", l.Name, err)
		}
		return nil
	}

	currentPythonVersion, err := Version(ctx)
	if err != nil {
		return err
	}
	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey,
		cache.WithFiles(reqs...),
		cache.WithStrings(currentPythonVersion))
	if err != nil {
		return err
	}

	// Check cache expiration to pick up new versions of dependencies that are not pinned.
	expired := cacheExpired(ctx, l)

	if cached && !expired {
		return nil
	}

	if expired {
		ctx.Debugf("Dependencies cache expired, clearing layer.")
	}

	if err := ctx.ClearLayer(l); err != nil {
		return fmt.Errorf("clearing layer %q: %w", l.Name, err)
	}

	ctx.Logf("Installing application dependencies.")
	cache.Add(ctx, l, dependencyHashKey, hash)
	// Update the layer metadata.
	ctx.SetMetadata(l, pythonVersionKey, currentPythonVersion)
	ctx.SetMetadata(l, expiryTimestampKey, time.Now().Add(expirationTime).Format(dateFormat))

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
		cmd := []string{
			"python3", "-m", "pip", "install",
			"--requirement", req,
			"--upgrade",
			"--upgrade-strategy", "only-if-needed",
			"--no-warn-script-location",   // bin is added at run time by lifecycle.
			"--no-warn-conflicts",         // Needed for python37 which allowed users to override dependencies. For newer versions, we do a separate `pip check`.
			"--force-reinstall",           // Some dependencies may be in the build image but not run image. Later requirements.txt should override earlier.
			"--no-compile",                // Prevent default timestamp-based bytecode compilation. Deterministic pycs are generated in a second step below.
			"--disable-pip-version-check", // If we were going to upgrade pip, we would have done it already in the runtime buildpack.
			"--no-cache-dir",              // We used to save this to a layer, but it made builds slower because it includes http caching of pypi requests.
		}
		vendorDir, isVendored := os.LookupEnv(VendorPipDepsEnv)
		if isVendored {
			cmd = append(cmd, "--no-index", "--find-links", vendorDir)
			buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.PipVendorDependenciesCounterID).Increment(1)
		}
		if !virtualEnv {
			cmd = append(cmd, "--user") // Install into user site-packages directory.
		}
		if _, err := ctx.Exec(cmd,
			gcp.WithUserAttribution); err != nil {
			return err
		}
	}

	// Generate deterministic hash-based pycs (https://www.python.org/dev/peps/pep-0552/).
	// Use the unchecked version to skip hash validation at run time (for faster startup).
	result, cerr := ctx.Exec([]string{
		"python3", "-m", "compileall",
		"--invalidation-mode", "unchecked-hash",
		"-qq", // Do not print any message (matches `pip install` behavior).
		l.Path,
	},
		gcp.WithUserAttribution)
	if cerr != nil {
		if result != nil {
			if result.ExitCode == 1 {
				// Ignore file compilation errors (matches `pip install` behavior).
				return nil
			}
			return fmt.Errorf("compileall: %s", result.Combined)
		}
		return fmt.Errorf("compileall: %v", cerr)
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
