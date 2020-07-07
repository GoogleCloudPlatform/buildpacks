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

// Package clearsource contains tools to delete source code.
package clearsource

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	defaultExclusions = []string{appengine.ConfigDir}
)

// DetectFn detemines if clear source buildpacks should opt in.
func DetectFn(ctx *gcp.Context) error {
	if devmode.Enabled(ctx) {
		ctx.OptOut("Development mode enabled")
	}

	if clearSource, ok := os.LookupEnv(env.ClearSource); ok {
		clear, err := strconv.ParseBool(clearSource)
		if err != nil {
			return fmt.Errorf("parsing %q: %v", env.ClearSource, err)
		}

		if clear {
			return nil
		}
	}
	ctx.OptOut("%s not set", env.ClearSource)
	return nil
}

// BuildFn clears the workspace while leaving exclusion patterns untouched.
// exclusions is a list of pattern strings relative to the user application directory.
func BuildFn(ctx *gcp.Context, exclusions []string) error {
	ctx.Logf("Clearing source")

	defer func(now time.Time) {
		ctx.Span("Clear source", now, gcp.StatusOk)
	}(time.Now())

	exclusions = append(exclusions, defaultExclusions...)
	paths, err := pathsToRemove(ctx, ctx.ApplicationRoot(), exclusions)
	if err != nil {
		return fmt.Errorf("filtering paths: %w", err)
	}
	for _, path := range paths {
		ctx.RemoveAll(path)
	}

	return nil
}

// pathsToRemove returns a list of entries in dir, filtering entries that match any in exclusions. exclusions should be a partial path relative to dir.
func pathsToRemove(ctx *gcp.Context, dir string, exclusions []string) ([]string, error) {
	paths := ctx.Glob(filepath.Join(dir, "*"))
	var filteredPaths []string
	for _, path := range paths {
		remove := true
		for _, exclusion := range exclusions {
			if match, err := filepath.Match(path, filepath.Join(dir, exclusion)); err != nil {
				return nil, fmt.Errorf("matching pattern %q with path %q: %v", filepath.Join(dir, exclusion), path, err)
			} else if match {
				remove = false
				break
			}
		}
		if remove {
			filteredPaths = append(filteredPaths, path)
		}
	}
	return filteredPaths, nil
}
