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

package gcpbuildpack

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// Glob is a pass through for filepath.Glob(...). It returns any error with proper user / system attribution.
func (ctx *Context) Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "globbing %s: %v", pattern, err)
	}
	return matches, nil
}

// HasAtLeastOne walks through file tree searching for at least one match.
func (ctx *Context) HasAtLeastOne(pattern string) (bool, error) {
	return ctx.HasAtLeastOneFiltered(pattern, nil)
}

// HasAtLeastOneOutsideDependencyDirectories walks through file tree searching
// for at least one match while ignoring dependency-only directories.
func (ctx *Context) HasAtLeastOneOutsideDependencyDirectories(pattern string) (bool, error) {
	filterFunc := func(path string) bool {
		rootedPath := "/" + path

		// Ignore the `node_modules` folder (as it may contain non-NodeJS files)
		return !strings.HasSuffix(rootedPath, "/node_modules")
	}

	return ctx.HasAtLeastOneFiltered(pattern, filterFunc)
}

type filepathFilter func(string) bool

// HasAtLeastOneFiltered is a pass through for filepath.Glob(...) it returns true if there is at least one
// file which matches the search pattern and is included by `filter`
func (ctx *Context) HasAtLeastOneFiltered(pattern string, filter filepathFilter) (bool, error) {
	dir := ctx.ApplicationRoot()

	errFileMatch := errors.New("File matched")
	matches, err := ctx.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return false, err
	}
	if len(matches) > 0 {
		return true, nil
	}

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if filter != nil && !filter(path) {
			return filepath.SkipDir
		}
		if err != nil {
			return buildererror.Errorf(buildererror.StatusInternal, "walking through %s within %s: %v", path, dir, err)
		}
		match, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return buildererror.Errorf(buildererror.StatusInternal, "matching %s with pattern %s: %v", path, pattern, err)
		}
		if match {
			return errFileMatch
		}
		return nil
	}); err != nil {
		if err == errFileMatch {
			return true, nil
		}
		return false, buildererror.Errorf(buildererror.StatusInternal, "walking through %s: %v", dir, err)
	}
	return false, nil
}
