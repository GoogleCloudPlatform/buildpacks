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
)

// Glob returns the names of all files matching pattern or nil if there is no matching file, exiting on any error.
func (ctx *Context) Glob(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "globbing %s: %v", pattern, err))
	}
	return matches
}

// HasAtLeastOne walks through file tree searching for at least one match.
func (ctx *Context) HasAtLeastOne(pattern string) bool {
	dir := ctx.ApplicationRoot()

	errFileMatch := errors.New("File matched")
	if len(ctx.Glob(filepath.Join(dir, pattern))) > 0 {
		return true
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ctx.Exit(1, Errorf(StatusInternal, "walking through %s within %s: %v", path, dir, err))
		}
		match, err := filepath.Match(pattern, info.Name())
		if err != nil {
			ctx.Exit(1, Errorf(StatusInternal, "matching %s with pattern %s: %v", path, pattern, err))
		}
		if match {
			return errFileMatch
		}
		return nil
	})
	if err == errFileMatch {
		return true
	}
	if err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "walking through %s: %v", dir, err))
	}
	return false
}
