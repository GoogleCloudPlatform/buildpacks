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

// Implements /bin/build for go/appengine_gopath buildpack.
// Package main sets $GOPATH then moves all gopath dependencies from _gopath/src/* to $GOPATH/src/*. The _gopath directory comes from go-app-stager.
// It then checks for _gopath/main-package-path which can only exist if the user's main package was originally on $GOPATH/src locally.
// If this file exists it moves the main package to $GOPATH/src and sets the path to build $GOPATH/src/<path-to-main-package> where <path-to-main-package> is read from _gopath/main-package-path.
// If this file doesn't exist it sets the path to build to "./...". Then it removes the _gopath directory because the build will fail if there's more than one go package in application root.
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if ctx.FileExists("go.mod") {
		ctx.OptOut("go.mod file found")
	}
	if !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.go") {
		ctx.OptOut("No *.go files found")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("gopath")

	goPath := l.Root
	goPathSrc := filepath.Join(goPath, "src")

	ctx.MkdirAll(goPathSrc, 0755)

	ctx.OverrideBuildEnv(l, "GOPATH", goPath)
	ctx.OverrideBuildEnv(l, "GO111MODULE", "off")
	ctx.WriteMetadata(l, nil, layers.Build)

	stagerGoPath := filepath.Join(ctx.ApplicationRoot(), "_gopath")
	stagerGoPathSrc := filepath.Join(stagerGoPath, "src")
	stagerGoPathMain := filepath.Join(stagerGoPath, "main-package-path")

	if ctx.FileExists(stagerGoPathSrc) {
		for _, f := range ctx.ReadDir(stagerGoPathSrc) {
			// To avoid superfluous files in root of stagerGoPathSrc, copy the subdirectories individually.
			if !f.IsDir() {
				continue
			}
			copyDir(ctx, filepath.Join(stagerGoPathSrc, f.Name()), filepath.Join(goPathSrc, f.Name()))
		}
	}

	var buildMainPath string
	if ctx.FileExists(stagerGoPathMain) {
		buildMainPath = filepath.Join(goPathSrc, strings.TrimSpace(string(ctx.ReadFile(stagerGoPathMain))))
		// Remove stager directory prior to copying to make sure we don't copy the stager directory to $GOPATH.
		ctx.RemoveAll(stagerGoPath)
		ctx.MkdirAll(buildMainPath, 0755)
		copyDir(ctx, ctx.ApplicationRoot(), buildMainPath)
	} else {
		buildMainPath = "./..."
		// Remove stager directory to make sure there's only one go package in application root.
		ctx.RemoveAll(stagerGoPath)
	}

	if _, exists := os.LookupEnv(env.Buildable); !exists {
		ctx.OverrideBuildEnv(l, env.Buildable, buildMainPath)
	}

	// TODO: cache the GOCACHE layer once you prove that the latency of re-dding a large layer is less than the latency of building without GOCACHE.

	return nil
}

func copyDir(ctx *gcp.Context, src, dst string) {
	ctx.Debugf("copying %q to %q", src, dst)

	// Trailing "/." copies the contents of src directory, but not src itself.
	src = filepath.Clean(src) + string(filepath.Separator) + "."
	ctx.Exec([]string{"cp", "--dereference", "-R", src, dst})
}
