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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// TempDir creates a directory with the provided name in the buildpack temp layer and returns its path
func (ctx *Context) TempDir(name string) (string, error) {
	tmpLayer := ctx.Layer("gcpbuildpack-tmp")
	directory := filepath.Join(tmpLayer.Path, name)
	if err := ctx.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	return directory, nil
}

// WriteFile is a pass through for ioutil.WriteFile(...) and returns any error with proper user / system attribution
func (ctx *Context) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := ioutil.WriteFile(filename, data, perm); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "writing file %q: %v", filename, err)
	}
	return nil
}

// ReadFile is a pass through for ioutil.ReadFile(...) and returns any error with proper user / system attribution
func (ctx *Context) ReadFile(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "reading file %q: %v", filename, err)
	}
	return data, nil
}

// ReadDir is a pass through for ioutil.ReadDir(...) and returns any error with proper user / system attribution
func (ctx *Context) ReadDir(elem ...string) ([]os.FileInfo, error) {
	n := filepath.Join(elem...)
	files, err := ioutil.ReadDir(n)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "reading directory %q: %v", n, err)
	}
	return files, nil
}
