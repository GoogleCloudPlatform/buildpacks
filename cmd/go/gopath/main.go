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

// Implements go/gopath buildpack.
// The gopath buildpack downloads dependencies with `go get`.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	goModExists, err := ctx.FileExists("go.mod")
	if err != nil {
		return nil, err
	}
	if goModExists {
		return gcp.OptOut("go.mod found"), nil
	}
	return gcp.OptIn("go.mod file not found, assuming GOPATH build"), nil
}

func buildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("gopath", gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating GOPATH layer: %w", err)
	}
	l.SharedEnvironment.Override("GOPATH", l.Path)
	l.SharedEnvironment.Override("GO111MODULE", "off")

	// TODO(b/145604612): Investigate caching the modules layer.

	// Skip 'go get' for Go versions >= 1.22 due to changes in module behavior
	if supportsGoGet, err := golang.SupportsGoGet(ctx); err != nil {
		return err
	} else if !supportsGoGet {
		ctx.Logf("\"go get\" has been skipped as go get is no longer supported outside of a module in the legacy GOPATH mode for go122+")

		goPath := l.Path
		goPathSrc := filepath.Join(goPath, "src")
		vendorPath := filepath.Join(ctx.ApplicationRoot(), "vendor")

		vendorPathExists, err := ctx.FileExists(vendorPath)
		if err != nil {
			return err
		}

		if vendorPathExists {

			err := fileutil.MaybeCopyPathContents(goPathSrc, vendorPath, fileutil.AllPaths)
			if err != nil {
				return err
			}
		}
		return nil
	}

	_, err = ctx.Exec([]string{"go", "get", "-d"}, gcp.WithEnv("GOPATH="+l.Path, "GO111MODULE=off"), gcp.WithUserAttribution)
	return err
}
