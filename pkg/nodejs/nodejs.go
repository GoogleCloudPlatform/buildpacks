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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/Masterminds/semver"
)

const (
	// EnvDevelopment represents a NODE_ENV development value.
	EnvDevelopment = "development"
	// EnvProduction represents a NODE_ENV production value.
	EnvProduction = "production"
	// EnvNodeVersion can be used to specify the version of Node.js is used for an app.
	EnvNodeVersion = "GOOGLE_NODEJS_VERSION"

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

// ReadPackageJSONIfExists returns deserialized package.json from the given dir. If the provided dir
// does not contain a package.json file it returns nil. Empty dir string uses the current working
// directory.
func ReadPackageJSONIfExists(dir string) (*PackageJSON, error) {
	f := filepath.Join(dir, "package.json")
	rawpjs, err := ioutil.ReadFile(f)
	if os.IsNotExist(err) {
		// Return an empty struct if the file doesn't exist (null object pattern).
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading package.json: %v", err)
	}

	var pjs PackageJSON
	if err := json.Unmarshal(rawpjs, &pjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling package.json: %v", err)
	}
	return &pjs, nil
}

// HasGCPBuild returns true if the given directory contains a package.json file that includes a
// non-empty "gcp-build" script.
func HasGCPBuild(dir string) (bool, error) {
	p, err := ReadPackageJSONIfExists(dir)
	if err != nil || p == nil {
		return false, err
	}
	return p.Scripts.GCPBuild != "", nil
}

// HasDevDependencies returns true if the given directory contains a package.json file that lists
// more one or more devDependencies.
func HasDevDependencies(dir string) (bool, error) {
	p, err := ReadPackageJSONIfExists(dir)
	if err != nil || p == nil {
		return false, err
	}
	return len(p.DevDependencies) > 0, nil
}

// RequestedNodejsVersion returns any customer provided Node.js version constraint by inspecting the
// environment and the package.json.
func RequestedNodejsVersion(ctx *gcp.Context, dir string) (string, error) {
	if version := os.Getenv(EnvNodeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", EnvNodeVersion, version)
		return version, nil
	}
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using runtime version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	pjs, err := ReadPackageJSONIfExists(dir)
	if err != nil || pjs == nil {
		return "", err
	}
	return pjs.Engines.Node, nil
}

// nodeVersion returns the installed version of Node.js.
// It can be overridden for testing.
var nodeVersion = func(ctx *gcp.Context) (string, error) {
	result, err := ctx.Exec([]string{"node", "-v"})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// isPreNode11 returns true if the installed version of Node.js is
// v10.x.x or older.
func isPreNode11(ctx *gcp.Context) (bool, error) {
	nodeVer, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(nodeVer)
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

// CheckOrClearCache checks whether cached dependencies exist and match. If they do not match, the
// layer is cleared and the layer metadata is updated with the new cache key.
func CheckOrClearCache(ctx *gcp.Context, l *libcnb.Layer, opts ...cache.Option) (bool, error) {
	currentNodeVersion, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
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
		ctx.CacheHit(l.Name)
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}

	ctx.CacheMiss(l.Name)
	if err := ctx.ClearLayer(l); err != nil {
		return false, fmt.Errorf("clearing layer: %v", err)
	}

	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, currentDependencyHash)
	ctx.SetMetadata(l, nodeVersionKey, currentNodeVersion)

	return false, nil
}

// SkipSyntaxCheck returns true if we should skip checking the user's function file for syntax errors
// if it is impacted by https://github.com/GoogleCloudPlatform/functions-framework-nodejs/issues/407.
func SkipSyntaxCheck(ctx *gcp.Context, file string) (bool, error) {
	nodeVer, err := nodeVersion(ctx)
	if err != nil {
		return false, err
	}
	version, err := semver.NewVersion(nodeVer)
	if err != nil {
		return false, gcp.InternalErrorf("failed to detect valid Node.js version %s: %v", version, err)
	}
	if version.Major() != 16 {
		return false, nil
	}
	if strings.HasSuffix(file, ".mjs") {
		return true, nil
	}
	pjs, err := ReadPackageJSONIfExists(ctx.ApplicationRoot())
	return (pjs != nil && pjs.Type == "module"), err
}

// IsNodeJS8Runtime returns true when the GOOGLE_RUNTIME is nodejs8. This will be
// true when using GCF or GAE with nodejs8. This function is useful for some
// legacy behavior in GCF.
func IsNodeJS8Runtime() bool {
	return os.Getenv(env.Runtime) == "nodejs8"
}
