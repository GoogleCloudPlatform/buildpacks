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
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

// Metadata represents metadata stored for a dependencies layer.
type Metadata struct {
	PythonVersion  string `toml:"python_version"`
	DependencyHash string `toml:"dependency_hash"`
}

// Version returns the installed version of Python.
func Version(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"python3", "--version"})
	return strings.TrimSpace(result.Stderr)
}

// CheckCache checks whether cached dependencies exist and match.
func CheckCache(ctx *gcp.Context, l *layers.Layer, files ...string) (bool, *Metadata, error) {
	currentPythonVersion := Version(ctx)
	currentDependencyHash, err := gcp.DependencyHash(ctx, currentPythonVersion, files...)
	if err != nil {
		return false, nil, fmt.Errorf("computing dependency hash: %v", err)
	}

	var meta Metadata
	ctx.ReadMetadata(l, &meta)

	// Perform install, skipping if the dependency hash matches existing metadata.
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", meta.DependencyHash)
	if currentDependencyHash == meta.DependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, &meta, nil
	}

	if meta.DependencyHash == "" {
		ctx.Logf("No metadata found or no dependency_hash found on metadata. Unable to use cache layer.")
	}
	ctx.Logf("Installing application dependencies.")
	// Update the layer metadata.
	meta.DependencyHash = currentDependencyHash
	meta.PythonVersion = currentPythonVersion

	return false, &meta, nil
}
