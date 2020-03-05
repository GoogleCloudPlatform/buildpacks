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

// Package nodejs contains Node.js buildpack library code.
package nodejs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

type packageEnginesJSON struct {
	Node string `json:"node"`
}

type packageScriptsJSON struct {
	Start    string `json:"start"`
	GCPBuild string `json:"gcp-build"`
}

// PackageJSON represents the contents of a package.json file.
type PackageJSON struct {
	Main            string             `json:"main"`
	Version         string             `json:"version"`
	Engines         packageEnginesJSON `json:"engines"`
	Scripts         packageScriptsJSON `json:"scripts"`
	Dependencies    map[string]string  `json:"dependencies"`
	DevDependencies map[string]string  `json:"devDependencies"`
}

// Metadata represents metadata stored for a dependencies layer.
type Metadata struct {
	NodeVersion    string `toml:"node_version"`
	DependencyHash string `toml:"dependency_hash"`
}

// ReadPackageJSON returns deserialized package.json from the given dir. Empty dir uses the current working directory.
func ReadPackageJSON(dir string) (*PackageJSON, error) {
	f := filepath.Join(dir, "package.json")
	rawpjs, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %v", err)
	}
	var pjs PackageJSON
	if err := json.Unmarshal(rawpjs, &pjs); err != nil {
		return nil, fmt.Errorf("unmarshalling package.json: %v", err)
	}
	return &pjs, nil
}

// NodeVersion returns the installed version of Node.js.
func NodeVersion(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"node", "-v"})
	return result.Stdout
}

// CheckCache checks whether cached dependencies exist and match.
func CheckCache(ctx *gcp.Context, l *layers.Layer, files ...string) (bool, *Metadata, error) {
	currentNodeVersion := NodeVersion(ctx)
	currentDependencyHash, err := gcp.DependencyHash(ctx, currentNodeVersion, files...)
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
	meta.NodeVersion = currentNodeVersion

	return false, &meta, nil
}
