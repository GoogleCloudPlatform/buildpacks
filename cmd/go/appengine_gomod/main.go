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

// Implements /bin/build for go/appengine_gomod buildpack.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	// stagerFileName is an optional file created by go-app-stager.
	// This file contains the main package path to build. This value can be overridden using GAE_YAML_MAIN.
	stagerFileName = "_main-package-path"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("go.mod") {
		ctx.OptOut("go.mod file not found")
	}

	if path, exists := os.LookupEnv(env.Buildable); exists {
		ctx.OptOut("%s already defined as %q", env.Buildable, path)
	}

	return nil
}

func buildFn(ctx *gcp.Context) error {
	buildMainPath, err := cleanMainPath(mainPath(ctx, ctx.ApplicationRoot()))
	if err != nil {
		return gcp.UserErrorf("cleaning main package path: %v", err)
	}

	if buildMainPath != "." {
		// If mainPath refers to a file, we prefix it with "./" so that `go build` treats it as such (in a later step).
		if ctx.FileExists(buildMainPath) {
			buildMainPath = "." + string(filepath.Separator) + buildMainPath
		} else {
			ctx.Logf("Path %q does not exist. Assuming it's a fully qualified package name.", buildMainPath)
		}
	}

	l := ctx.Layer("main_env")
	ctx.OverrideBuildEnv(l, env.Buildable, buildMainPath)
	ctx.WriteMetadata(l, nil, layers.Build)

	return nil
}

// mainPath chooses the main package path from the paths provided by _main-package-path or GAE_YAML_MAIN.
func mainPath(ctx *gcp.Context, appRoot string) string {
	if path := os.Getenv(env.GAEMain); path != "" {
		return path
	}

	if pathFile := filepath.Join(appRoot, stagerFileName); ctx.FileExists(pathFile) {
		path := string(ctx.ReadFile(pathFile))
		ctx.RemoveAll(pathFile)
		return path
	}

	return ""
}

func cleanMainPath(mp string) (string, error) {
	mp = filepath.Clean(filepath.ToSlash(strings.TrimSpace(mp)))
	if mp == "." {
		return ".", nil
	}
	if filepath.IsAbs(mp) {
		return "", fmt.Errorf("main package path %q must not be absolute path", mp)
	}
	if strings.HasPrefix(mp, "..") {
		return "", fmt.Errorf("main package path %q cannot reference parent", mp)
	}
	return mp, nil
}
