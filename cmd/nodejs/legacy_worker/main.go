// Copyright 2021 Google LLC
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

// Implements nodejs/legacy-worker buildpack.
// The legacy-worker buildpack converts a function into an application and sets up the execution environment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName = "legacy-worker"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !nodejs.IsNodeJS8Runtime() {
		return gcp.OptOut("Only compatible with nodejs8"), nil
	}
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

// buildFn sets up the execution environment for the function.
// For a function that specifies the framework as a dependency, only set
// environment variables and define a web process. The framework is
// installed in the npm or yarn buildpack with other dependencies.
// For a function that does not, also install the framework.
func buildFn(ctx *gcp.Context) error {

	if _, ok := os.LookupEnv(env.FunctionSource); ok {
		return gcp.UserErrorf("%s is not currently supported for Node.js buildpacks", env.FunctionSource)
	}

	// Function source code should be defined in the "main" field in package.json, index.js or function.js.
	// https://cloud.google.com/functions/docs/writing#structuring_source_code
	fnFile := "function.js"
	indexJSExists, err := ctx.FileExists("index.js")
	if err != nil {
		return err
	}
	if indexJSExists {
		fnFile = "index.js"
	}
	pjs, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
	if err != nil {
		return err
	}
	if pjs != nil && pjs.Main != "" {
		fnFile = pjs.Main
	}

	fnFileExists, err := ctx.FileExists(fnFile)
	if err != nil {
		return err
	}
	if !fnFileExists {
		return gcp.UserErrorf("%s does not exist", fnFile)
	}

	// Syntax check the function code without executing to prevent run-time errors.
	if _, err := ctx.Exec([]string{"node", "--check", fnFile}, gcp.WithUserAttribution); err != nil {
		return err
	}

	l, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}

	if err := installLegacyWorker(ctx, l); err != nil {
		return fmt.Errorf("installing worker.js: %w", err)
	}

	nm := filepath.Join(ctx.ApplicationRoot(), "node_modules")
	nmExists, err := ctx.FileExists(nm)
	if err != nil {
		return err
	}
	// The environment variables required by worker.js are different than those expected
	// by the Functions Frameworks (hence we don't use ctx.SetFunctionsEnvVars()).

	// Add user's node_modules to NODE_PATH so functions-framework can always find user's packages.
	if nmExists {
		l.LaunchEnvironment.Prepend("NODE_PATH", string(os.PathListSeparator), nm)
	}
	if target := os.Getenv(env.FunctionTarget); target != "" {
		l.LaunchEnvironment.Default("X_GOOGLE_FUNCTION_NAME", target)
		l.LaunchEnvironment.Default("X_GOOGLE_ENTRY_POINT", target)
	} else {
		// This should never happen because this env var is used by the detect phase.
		return gcp.InternalErrorf("required env var %s not found", env.FunctionTarget)
	}
	signature := os.Getenv(env.FunctionSignatureType)
	if signature == "http" || signature == "" {
		// The name of the HTTP signature type is slightly different for worker.js
		// than that of Functions Frameworks.
		signature = "HTTP_TRIGGER"
	}
	l.LaunchEnvironment.Default("X_GOOGLE_FUNCTION_TRIGGER_TYPE", signature)
	l.LaunchEnvironment.Default("X_GOOGLE_CODE_LOCATION", ctx.ApplicationRoot())

	// TODO(b/184077805) this can be removed after the corresponding code from worker.js is removed
	l.LaunchEnvironment.Default("X_GOOGLE_NEW_FUNCTION_SIGNATURE", "true")
	// TODO(b/184077805) default to 8080 match FF runtimes?
	l.LaunchEnvironment.Default("X_GOOGLE_WORKER_PORT", 8091)
	l.LaunchEnvironment.Default("WORKER_PORT", 8091)

	// TODO(b/181987135) historically worker.js was run with the --max-old-space-size to set the heap
	// size. We should replicate this behaviour via the NODE_OPTIONS env var.
	worker := filepath.Join(l.Path, "worker.js")
	ctx.AddWebProcess([]string{"node", worker})
	return nil
}

// installLegacyWorker copies worker.js and installs its dependencies in the given layer.
func installLegacyWorker(ctx *gcp.Context, l *libcnb.Layer) error {
	ctx.Logf("Configuring the legacy Google Cloud Functions worker.js.")

	cvt := filepath.Join(ctx.BuildpackRoot(), "converter", "worker")
	pjs := filepath.Join(cvt, "package.json")
	wjs := filepath.Join(cvt, "worker.js")

	cached, err := nodejs.CheckOrClearCache(ctx, l, cache.WithStrings(nodejs.EnvProduction), cache.WithFiles(pjs, wjs))
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		return nil
	}
	ctx.Logf("Installing worker dependencies.")
	installCmd, err := nodejs.NPMInstallCommand(ctx)
	if err != nil {
		return err
	}

	if _, err := ctx.Exec([]string{"cp", "-t", l.Path, pjs, wjs}, gcp.WithUserTimingAttribution); err != nil {
		return err
	}
	if _, err := ctx.Exec([]string{"npm", installCmd, "--quiet", "--production", "--prefix", l.Path}, gcp.WithUserAttribution); err != nil {
		return err
	}
	return nil
}
