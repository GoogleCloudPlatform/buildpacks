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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/Masterminds/semver"
)

const (
	// EnvDevelopment represents a NODE_ENV development value.
	EnvDevelopment = "development"
	// EnvProduction represents a NODE_ENV production value.
	EnvProduction = "production"

	nodeVersionKey    = "node_version"
	dependencyHashKey = "dependency_hash"
)

// semVer11 is the smallest possible semantic version with major version 11.
var semVer11 = semver.MustParse("11.0.0")

type packageEnginesJSON struct {
	Node string `json:"node"`
	NPM  string `json:"npm"`
	Yarn string `json:"yarn"`
}

type packageScriptsJSON struct {
	Start    string `json:"start"`
	GCPBuild string `json:"gcp-build"`
}

// PackageJSON represents the contents of a package.json file.
type PackageJSON struct {
	Main            string             `json:"main"`
	Type            string             `json:"type"`
	Version         string             `json:"version"`
	Engines         packageEnginesJSON `json:"engines"`
	Scripts         packageScriptsJSON `json:"scripts"`
	Dependencies    map[string]string  `json:"dependencies"`
	DevDependencies map[string]string  `json:"devDependencies"`
}

// ReadPackageJSON returns deserialized package.json from the given dir. Empty dir uses the current working directory.
func ReadPackageJSON(dir string) (*PackageJSON, error) {
	f := filepath.Join(dir, "package.json")
	rawpjs, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, gcp.InternalErrorf("reading package.json: %v", err)
	}
	var pjs PackageJSON
	if err := json.Unmarshal(rawpjs, &pjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling package.json: %v", err)
	}
	return &pjs, nil
}

// nodeVersion returns the installed version of Node.js.
// It can be overridden for testing.
var nodeVersion = func(ctx *gcp.Context) string {
	result := ctx.Exec([]string{"node", "-v"})
	return result.Stdout
}

// isPreNode11 returns true if the installed version of Node.js is
// v10.x.x or older.
func isPreNode11(ctx *gcp.Context) (bool, error) {
	version, err := semver.NewVersion(nodeVersion(ctx))
	if err != nil {
		return false, gcp.InternalErrorf("failed to detect valid Node.js version %s: %v", version, err)
	}
	return version.LessThan(semVer11), nil
}

// NodeEnv returns the value of NODE_ENV or `production`.
func NodeEnv() string {
	nodeEnv := os.Getenv("NODE_ENV")
	if nodeEnv == "" {
		nodeEnv = EnvProduction
	}
	return nodeEnv
}

// CheckCache checks whether cached dependencies exist and match.
func CheckCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentNodeVersion := nodeVersion(ctx)
	opts = append(opts, cache.WithStrings(currentNodeVersion))
	currentDependencyHash, err := cache.Hash(ctx, opts...)
	if err != nil {
		return false, fmt.Errorf("computing dependency hash: %v", err)
	}

	// Perform install, skipping if the dependency hash matches existing metadata.
	metaDependencyHash := ctx.GetMetadata(l, dependencyHashKey)
	ctx.Debugf("Current dependency hash: %q", currentDependencyHash)
	ctx.Debugf("  Cache dependency hash: %q", metaDependencyHash)
	if currentDependencyHash == metaDependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	ctx.Logf("Installing application dependencies.")

	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, currentDependencyHash)
	ctx.SetMetadata(l, nodeVersionKey, currentNodeVersion)

	return false, nil
}

// SkipSyntaxCheck returns true if we should skip checking the user's function file for syntax errors
// if it is impacted by https://github.com/GoogleCloudPlatform/functions-framework-nodejs/issues/407.
func SkipSyntaxCheck(ctx *gcp.Context, file string) (bool, error) {
	version, err := semver.NewVersion(nodeVersion(ctx))
	if err != nil {
		return false, gcp.InternalErrorf("failed to detect valid Node.js version %s: %v", version, err)
	}
	if version.Major() != 16 {
		return false, nil
	}
	if strings.HasSuffix(file, ".mjs") {
		return true, nil
	}
	if !ctx.FileExists(filepath.Join(ctx.ApplicationRoot(), "package.json")) {
		return false, nil
	}
	pjs, err := ReadPackageJSON(ctx.ApplicationRoot())
	return (pjs != nil && pjs.Type == "module"), err
}
