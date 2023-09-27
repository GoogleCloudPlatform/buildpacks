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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// TempDir creates a directory with the provided name in the buildpack temp layer and returns its path
func (ctx *Context) TempDir(name string) (string, error) {
	tmpLayer, err := ctx.Layer("gcpbuildpack-tmp")
	if err != nil {
		return "", fmt.Errorf("creating layer: %w", err)
	}
	directory := filepath.Join(tmpLayer.Path, name)
	if err := ctx.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	return directory, nil
}

// WriteFile is a pass through for os.WriteFile(...) and returns any error with proper user / system attribution
func (ctx *Context) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return buildererror.Errorf(buildererror.StatusInternal, "writing file %q: %v", filename, err)
	}
	return nil
}

// ReadFile is a pass through for os.ReadFile(...) and returns any error with proper user / system attribution
func (ctx *Context) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "reading file %q: %v", filename, err)
	}
	return data, nil
}

// ReadDir is a pass through for os.ReadDir(...) and returns any error with proper user / system attribution
func (ctx *Context) ReadDir(elem ...string) ([]os.FileInfo, error) {
	n := filepath.Join(elem...)
	entries, err := os.ReadDir(n)
	if err != nil {
		return nil, buildererror.Errorf(buildererror.StatusInternal, "reading directory %q: %v", n, err)
	}
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, buildererror.Errorf(buildererror.StatusInternal, "reading directory %q: %v", n, err)
		}
		infos = append(infos, info)
	}

	return infos, nil
}
