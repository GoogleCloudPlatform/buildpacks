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

// Implements /bin/build for nodejs/functions-framework buildpack.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName = "functions-framework"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); !ok {
		ctx.OptOut("%s not set.", env.FunctionTarget)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	// Function source code should be defined in the "main" field in package.json, index.js or function.js.
	// https://cloud.google.com/functions/docs/writing#structuring_source_code
	fnFile := "function.js"
	if ctx.FileExists("index.js") {
		fnFile = "index.js"
	}

	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	if ctx.FileExists("package.json") {
		pjs, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
		if err != nil {
			return gcp.Errorf(gcp.StatusInvalidArgument, "reading package.json in %q: %v", ctx.ApplicationRoot(), err)
		}
		_, hasFrameworkDependency = pjs.Dependencies["@google-cloud/functions-framework"]
		if pjs.Main != "" {
			fnFile = pjs.Main
		}
	}

	ctx.ExecUser([]string{"node", "--check", fnFile})

	cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
	if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		cvt = filepath.Join(cvt, "with-framework")
	} else {
		ctx.Logf("Handling functions without dependency on functions-framework.")
		cvt = filepath.Join(cvt, "without-framework")
	}

	// Install functions-framework.
	l := ctx.Layer(layerName)
	nm := path.Join(l.Root, "node_modules")
	pjs := filepath.Join(cvt, "package.json")
	pljs := filepath.Join(cvt, nodejs.PackageLock)

	cached, meta, err := nodejs.CheckCache(ctx, l, pjs, pljs)
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)
		ctx.ClearLayer(l)
		// NPM expects package.json and the lock file in the prefix directory.
		ctx.Exec([]string{"cp", "-t", l.Root, pjs, pljs})
		cmd, err := nodejs.NPMInstallCommand(ctx)
		if err != nil {
			return fmt.Errorf("generating npm command: %w", err)
		}
		ctx.ExecUser([]string{"npm", cmd, "--quiet", "--production", "--prefix", l.Root})
	}

	// Determine the path to the executable file to start functions-framework.
	// If the function has dependency on functions-framework, it should already
	// be installed in node_modules.
	// Else, it is installed in functions-framework layer's node_modules.
	// Note that we DON'T add functions-framework layer's node_modules
	// to NODE_PATH because we don't want the function to load packages in
	// functions-framework layer's node_modules.
	ff := filepath.Join(".bin", "functions-framework")
	if hasFrameworkDependency {
		ff = filepath.Join("node_modules", ff)
	} else {
		ff = filepath.Join(nm, ff)
	}

	if err := env.SetFunctionsEnvVars(ctx, l); err != nil {
		return fmt.Errorf("setting functions env vars: %w", err)
	}

	ctx.AddWebProcess([]string{ff})
	ctx.WriteMetadata(l, &meta, layers.Build, layers.Cache, layers.Launch)

	return nil
}
