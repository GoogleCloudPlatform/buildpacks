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

// Implements go/appengine_gomod buildpack.
// The appengine_gomod buildpack sets up the path of the package to build for gomod applications.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

const (
	// stagerFileName is an optional file created by go-app-stager.
	// This file contains the main package path to build. This value can be overridden using GAE_YAML_MAIN.
	stagerFileName = "_main-package-path"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	goModExists, err := ctx.FileExists("go.mod")
	if err != nil {
		return nil, err
	}
	if !goModExists {
		return gcp.OptOutFileNotFound("go.mod"), nil
	}

	if path, exists := os.LookupEnv(env.Buildable); exists {
		return gcp.OptOut(fmt.Sprintf("%s already defined as %q", env.Buildable, path)), nil
	}

	return gcp.OptIn(fmt.Sprintf("found go.mod and %s is not set", env.Buildable)), nil
}

func buildFn(ctx *gcp.Context) error {
	mp, err := mainPath(ctx)
	if err != nil {
		return fmt.Errorf("choosing main path: %w", err)
	}
	buildMainPath, err := cleanMainPath(mp)
	if err != nil {
		return fmt.Errorf("cleaning main package path: %w", err)
	}

	if buildMainPath != "." {
		// If mainPath refers to a file, we prefix it with "./" so that `go build` treats it as such (in a later step).
		buildMainExists, err := ctx.FileExists(buildMainPath)
		if err != nil {
			return err
		}
		if buildMainExists {
			buildMainPath = "." + string(filepath.Separator) + buildMainPath
		} else {
			ctx.Logf("Path %q does not exist. Assuming it's a fully qualified package name.", buildMainPath)
		}
	}

	l := ctx.Layer("main_env", gcp.BuildLayer)
	l.BuildEnvironment.Override(env.Buildable, buildMainPath)

	// HACK: For backwards compatibility on App Engine Go 1.11:
	// Copy all files to a layer directory that ends with /srv because the appengine package relies on the name:
	// https://github.com/golang/appengine/blob/553959209a20f3be281c16dd5be5c740a893978f/delay/delay.go#L136
	// We change the work directory instead of modifying env.Buildable, because the latter is not necessarily a filesystem path.
	srvl := ctx.Layer("srv", gcp.BuildLayer)
	srvl.BuildEnvironment.Override(golang.BuildDirEnv, srvl.Path)
	ctx.Exec([]string{"cp", "--dereference", "-R", ".", srvl.Path}, gcp.WithUserTimingAttribution)

	return nil
}

// mainPath chooses the main package path from the paths provided by _main-package-path or GAE_YAML_MAIN.
func mainPath(ctx *gcp.Context) (string, error) {
	if path := os.Getenv(env.GAEMain); path != "" {
		return path, nil
	}

	pathFile := filepath.Join(ctx.ApplicationRoot(), stagerFileName)
	pathExists, err := ctx.FileExists(pathFile)
	if err != nil {
		return "", err
	}
	if pathExists {
		bytes, err := ctx.ReadFile(pathFile)
		if err != nil {
			return "", err
		}
		path := string(bytes)
		ctx.RemoveAll(pathFile)
		return path, nil
	}

	return "", nil
}

func cleanMainPath(mp string) (string, error) {
	mp = filepath.Clean(filepath.ToSlash(strings.TrimSpace(mp)))
	if mp == "." {
		return ".", nil
	}
	if filepath.IsAbs(mp) {
		return "", gcp.UserErrorf("main package path %q must not be absolute path", mp)
	}
	if strings.HasPrefix(mp, "..") {
		return "", gcp.UserErrorf("main package path %q cannot reference parent", mp)
	}
	return mp, nil
}
