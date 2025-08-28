// Copyright 2022 Google LLC
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

// Implements python/link-runtime buildpack.
// The link-runtime replaces the python layer content installed by the python/runtime buildpack
// with symlinks to the python installed in the GAE base images.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/Masterminds/semver"
)

const (
	// layerDir is the file path the python/runtime buildpack installs into.
	layerDir = "/layers/google.python.runtime/python"
)

// linkDirs is the set of directories that should be symlinked to the base image Python.
var linkDirs = []string{
	"bin",
	"include",
	"lib",
	"share",
}

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	skip, err := env.IsPresentAndTrue(env.XGoogleSkipRuntimeLaunch)
	if err != nil {
		return nil, err
	}
	if !skip {
		return gcp.OptOut(fmt.Sprintf("%s is not 'true'", env.XGoogleSkipRuntimeLaunch)), nil
	}
	if result := runtime.CheckOverride("python"); result != nil {
		return result, nil
	}
	return gcp.OptOut("GOOGLE_RUNTIME env var not a python runtime"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	pythonPath, err := pythonSystemDir(ctx)
	if err != nil {
		return err
	}
	for _, file := range linkDirs {
		p := filepath.Join(layerDir, file)
		if err := os.RemoveAll(p); err != nil {
			return gcp.InternalErrorf("removing %q: %w", p, err)
		}
		link := filepath.Join(pythonPath, file)
		if err := os.Symlink(link, p); err != nil {
			return gcp.InternalErrorf("creating symlink %s to %s: %w", link, p, err)
		}
	}
	return nil
}

// pythonSystemDir returns the file path that python is installed to in the GAE base images.
func pythonSystemDir(ctx *gcp.Context) (string, error) {
	ver, err := python.Version(ctx)
	if err != nil {
		return "", gcp.InternalErrorf("getting python version: %w", err)
	}
	trimmedVer := versionWithoutRCSuffix(strings.TrimPrefix(ver, "Python "))
	semver, err := semver.NewVersion(trimmedVer)
	if err != nil {
		return "", gcp.InternalErrorf("parsing python version %q: %w", ver, err)
	}
	return filepath.Join("/opt", fmt.Sprintf("python%d.%d", semver.Major(), semver.Minor())), nil
}

// Supporting RC candidate as an interim for Alpha release until final release is out.
// Example RC Candidate - 3.12.0rc1
func versionWithoutRCSuffix(version string) string {
	m := regexp.MustCompile("rc(.*)")
	return m.ReplaceAllString(version, "")
}
