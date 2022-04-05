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

// Implements go/appengine_gopath buildpack.
// The appengine_gopath buildpack sets $GOPATH and moves all gopath dependencies from _gopath/src/* to $GOPATH/src/*. The _gopath directory is created by go-app-stager during deployment.
// It then checks for _gopath/main-package-path which exists if the user's main package was originally on $GOPATH/src locally.
// If this file exists, the buildpack moves the main package to $GOPATH/src and sets the path to build $GOPATH/src/<path-to-main-package> where <path-to-main-package> is read from _gopath/main-package-path.
// If this file doesn't exist, the buildpack sets the path to build to "./..." and removes the _gopath directory because the build will fail if there's more than one go package in application root.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsGAE() {
		return appengine.OptOutTargetPlatformNotAE(), nil
	}
	goModExists, err := ctx.FileExists("go.mod")
	if err != nil {
		return nil, err
	}
	if goModExists {
		return gcp.OptOut("go.mod found"), nil
	}
	atLeastOne, err := ctx.HasAtLeastOne("*.go")
	if err != nil {
		return nil, fmt.Errorf("finding *.go files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .go files found"), nil
	}
	return gcp.OptIn("go.mod file not found, assuming GOPATH build"), nil
}

func buildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("gopath", gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating gopath layer: %w", err)
	}

	goPath := l.Path
	goPathSrc := filepath.Join(goPath, "src")

	if err := ctx.MkdirAll(goPathSrc, 0755); err != nil {
		return err
	}

	l.BuildEnvironment.Override("GOPATH", goPath)
	l.BuildEnvironment.Override("GO111MODULE", "off")

	stagerGoPath := filepath.Join(ctx.ApplicationRoot(), "_gopath")
	stagerGoPathSrc := filepath.Join(stagerGoPath, "src")
	stagerGoPathMain := filepath.Join(stagerGoPath, "main-package-path")

	stagerGoPathSrcExists, err := ctx.FileExists(stagerGoPathSrc)
	if err != nil {
		return err
	}
	if stagerGoPathSrcExists {
		files, err := ctx.ReadDir(stagerGoPathSrc)
		if err != nil {
			return err
		}
		for _, f := range files {
			// To avoid superfluous files in root of stagerGoPathSrc, copy the subdirectories individually.
			if !f.IsDir() {
				continue
			}
			copyDir(ctx, filepath.Join(stagerGoPathSrc, f.Name()), filepath.Join(goPathSrc, f.Name()))
		}
	}

	var buildMainPath string
	stagerGoPathMainExists, err := ctx.FileExists(stagerGoPathMain)
	if err != nil {
		return err
	}
	if stagerGoPathMainExists {
		goPathMainBytes, err := ctx.ReadFile(stagerGoPathMain)
		if err != nil {
			return err
		}
		buildMainPath = filepath.Join(goPathSrc, strings.TrimSpace(string(goPathMainBytes)))
		// Remove stager directory prior to copying to make sure we don't copy the stager directory to $GOPATH.
		if err := ctx.RemoveAll(stagerGoPath); err != nil {
			return err
		}
		if err := ctx.MkdirAll(buildMainPath, 0755); err != nil {
			return err
		}
		copyDir(ctx, ctx.ApplicationRoot(), buildMainPath)
	} else {
		buildMainPath = "./..."
		// Remove stager directory to make sure there's only one go package in application root.
		if err := ctx.RemoveAll(stagerGoPath); err != nil {
			return err
		}
	}

	if _, exists := os.LookupEnv(env.Buildable); !exists {
		l.BuildEnvironment.Override(env.Buildable, buildMainPath)
	}

	// Unlike in the appengine_gomod buildpack, we do not have to compile gopath apps from a path that ends in /srv/. There are two cases:
	//  * _gopath/main-package-path exists and app source is put on GOPATH, which is handled by:
	//			https://github.com/golang/appengine/blob/553959209a20f3be281c16dd5be5c740a893978f/delay/delay.go#L136.
	//  * _gopath/main-package-path does not exist and the app is built from the current directory, which is handled by:
	//			https://github.com/golang/appengine/blob/553959209a20f3be281c16dd5be5c740a893978f/delay/delay.go#L125-L127

	// TODO(b/145608768): Investigate creating and caching a GOCACHE layer.
	return nil
}

func copyDir(ctx *gcp.Context, src, dst string) {
	// Trailing "/." copies the contents of src directory, but not src itself.
	src = filepath.Clean(src) + string(filepath.Separator) + "."
	ctx.Exec([]string{"cp", "--dereference", "-R", src, dst}, gcp.WithUserTimingAttribution)
}
