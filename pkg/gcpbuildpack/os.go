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
	"os"
	"path/filepath"
)

// Rename renames the old path to the new path, exiting on any error.
func (ctx *Context) Rename(old, new string) {
	ctx.Debugf("Renaming %q to %q", old, new)
	if err := os.Rename(old, new); err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "renaming %s to %s: %v", old, new, err))
	}
}

// CreateFile creates the specified file, return the File object, exiting on any error.
func (ctx *Context) CreateFile(file string) *os.File {
	f, err := os.Create(file)
	if err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "creating %s: %v", file, err))
	}
	return f
}

// MkdirAll creates all necessary directories for the given path, exiting on any error.
func (ctx *Context) MkdirAll(path string, perm os.FileMode) {
	if err := os.MkdirAll(path, perm); err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "creating %s: %v", path, err))
	}
}

// RemoveAll removes the given path, exiting on any error.
func (ctx *Context) RemoveAll(elem ...string) {
	path := filepath.Join(elem...)
	if err := os.RemoveAll(path); err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "removing %s: %v", path, err))
	}
}

// Symlink creates newname as a symbolic name to oldname, exiting on any error.
func (ctx *Context) Symlink(oldname string, newname string) {
	if err := os.Symlink(oldname, newname); err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "symlinking from %q to %q: %v", oldname, newname, err))
	}
}

// FileExists returns true if a file exists at the path joined by elem, exiting on any error.
func (ctx *Context) FileExists(elem ...string) bool {
	path := filepath.Join(elem...)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else if err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "stat %q: %v", path, err))
	}
	return true
}

// IsWritable returns true if the file at the path constructed by joining elem is writable by the owner, exiting on any error.
func (ctx *Context) IsWritable(elem ...string) bool {
	path := filepath.Join(elem...)
	info, err := os.Stat(path)
	if err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "stat %q: %v", path, err))
	}
	return info.Mode().Perm()&0200 != 0
}

// Setenv immediately sets an environment variable, exiting on any error.
// Note: this only sets an env var for the current script invocation. If you need an env var that
// persists through the build environment or the launch environment, use ctx.PrependBuildEnv,...
func (ctx *Context) Setenv(key, value string) {
	ctx.Debugf("Setting environment variable %s=%s", key, value)
	if err := os.Setenv(key, value); err != nil {
		ctx.Exit(1, Errorf(StatusInternal, "setting env var %s: %v", key, err))
	}
}
