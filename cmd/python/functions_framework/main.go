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

// Implements python/functions_framework buildpack.
// The functions_framework buildpack converts a functionn into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName = "functions-framework"
)

var (
	ffRegexp  = regexp.MustCompile(`(?m)^functions-framework\b([^-]|$)`)
	eggRegexp = regexp.MustCompile(`(?m)#egg=functions-framework$`)
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		ctx.OptIn("%s set", env.FunctionTarget)
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	// Determine if the function has dependency on functions-framework.
	hasFrameworkDependency := false
	if ctx.FileExists("requirements.txt") {
		content := ctx.ReadFile("requirements.txt")
		hasFrameworkDependency = containsFF(string(content))
	}

	// Install functions-framework.
	l := ctx.Layer(layerName)
	if hasFrameworkDependency {
		ctx.Logf("Handling functions with dependency on functions-framework.")
		ctx.ClearLayer(l)

		// With framework dependency, framework module is in pip buildpack, so only env vars are present in this layer.
		ctx.WriteMetadata(l, nil, layers.Launch)
	} else {
		ctx.Logf("Handling functions without dependency on functions-framework.")
		if err := installFramework(ctx, l); err != nil {
			return fmt.Errorf("installing framework: %v", err)
		}
	}

	ctx.SetFunctionsEnvVars(l)

	ctx.ExecUser([]string{"python3", "-m", "compileall", "."})
	ctx.AddWebProcess([]string{"functions-framework"})
	return nil
}

func containsFF(s string) bool {
	return ffRegexp.MatchString(s) || eggRegexp.MatchString(s)
}

func installFramework(ctx *gcp.Context, l *layers.Layer) error {
	cvt := filepath.Join(ctx.BuildpackRoot(), "converter")
	req := filepath.Join(cvt, "requirements.txt")
	cached, meta, err := python.CheckCache(ctx, l, req)
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)
		ctx.ExecUser([]string{"python3", "-m", "pip", "install", "--upgrade", "-t", l.Root, "-r", req})
	}
	ctx.PrependPathSharedEnv(l, "PYTHONPATH", l.Root)
	ctx.WriteMetadata(l, &meta, layers.Build, layers.Cache, layers.Launch)
	return nil
}
