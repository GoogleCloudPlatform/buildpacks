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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// Rename is a pass through for os.Rename(...) and logs an informational statement and returns any error with proper user / system attribution
func (ctx *Context) Rename(old, new string) error {
	ctx.Debugf("Renaming %q to %q", old, new)
	if err := os.Rename(old, new); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "renaming %s to %s: %v", old, new, err)
	}
	return nil
}

// CreateFile is a pass through for os.Create(...) and returns any error with proper user / system attribution
func (ctx *Context) CreateFile(file string) (*os.File, error) {
	f, err := os.Create(file)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "creating %s: %v", file, err)
	}
	return f, nil
}

// MkdirAll is a pass through for os.Create(...) and returns any error with proper user / system attribution
func (ctx *Context) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "creating %s: %v", path, err)
	}
	return nil
}

// RemoveAll is a pass through for os.RemoveAll(...) and returns any error with proper user / system attribution
func (ctx *Context) RemoveAll(elem ...string) error {
	path := filepath.Join(elem...)
	if err := os.RemoveAll(path); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "removing %s: %v", path, err)
	}
	return nil
}

// Symlink is a pass through for os.Symlink(...) and returns any error with proper user / system attribution
func (ctx *Context) Symlink(oldname string, newname string) error {
	if err := os.Symlink(oldname, newname); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "symlinking from %q to %q: %v", oldname, newname, err)
	}
	return nil
}

// FileExists returns true if a file exists at the path joined by elem
func (ctx *Context) FileExists(elem ...string) (bool, error) {
	path := filepath.Join(elem...)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, buildererror.Errorf(buildererror.StatusInternal, "stat %q: %v", path, err)
	}
	return true, nil
}

// IsWritable returns true if the file at the path constructed by joining elem is writable by the owner
func (ctx *Context) IsWritable(elem ...string) (bool, error) {
	path := filepath.Join(elem...)
	info, err := os.Stat(path)
	if err != nil {
		return false, buildererror.Errorf(buildererror.StatusInternal, "stat %q: %v", path, err)
	}
	return info.Mode().Perm()&0200 != 0, nil
}

// Setenv is a pass through for os.Setenv(...) and returns any error with proper user / system attribution
// Note: this only sets an env var for the current script invocation. If you need an env var that
// persists through the build environment or the launch environment, use ctx.PrependBuildEnv,...
func (ctx *Context) Setenv(key, value string) error {
	ctx.Logf("Setting environment variable %s=%s", key, value)
	if err := os.Setenv(key, value); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "setting env var %s: %v", key, err)
	}
	return nil
}

// HomeDir returns the path of the $USER's $HOME directory.
func (ctx *Context) HomeDir() string {
	return os.Getenv("HOME")
}
