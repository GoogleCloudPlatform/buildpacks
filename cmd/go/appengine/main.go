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

// Implements go/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package main

import (
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsGAE() {
		return appengine.OptInTargetPlatformGAE(), nil
	}
	return appengine.OptOutTargetPlatformNotGAE(), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := validateAppEngineAPIs(ctx); err != nil {
		return err
	}
	return appengine.Build(ctx, "go", entrypoint)
}

func validateAppEngineAPIs(ctx *gcp.Context) error {
	supportsApis, err := golang.SupportsAppEngineApis(ctx)
	if err != nil {
		return err
	}

	dirDeps, err := directDeps(ctx)
	if err != nil {
		return err
	}
	if !supportsApis && appEngineInDeps(dirDeps) {
		// TODO(b/179431689) Change to error.
		ctx.Warnf(appengine.DepWarning)
		return nil
	}

	deps, err := allDeps(ctx)
	if err != nil {
		return err
	}
	usingAppEngine := appEngineInDeps(deps)
	if supportsApis && !usingAppEngine {
		ctx.Warnf(appengine.UnusedAPIWarning)
	}

	if !supportsApis && usingAppEngine {
		ctx.Warnf(appengine.IndirectDepWarning)
	}

	return nil
}

func entrypoint(ctx *gcp.Context) (*appstart.Entrypoint, error) {
	ctx.Logf("No user entrypoint specified. Using the generated entrypoint %q", golang.OutBin)
	return &appstart.Entrypoint{Type: appstart.EntrypointGenerated.String(), Command: golang.OutBin}, nil
}

func appEngineInDeps(deps []string) bool {
	for _, s := range deps {
		if strings.HasPrefix(s, "google.golang.org/appengine") {
			return true
		}
	}
	return false
}

func allDeps(ctx *gcp.Context) ([]string, error) {
	result, err := ctx.Exec([]string{"go", "list", "-e", "-f", `{{join .Deps "\n"}}`, "./..."}, gcp.WithUserAttribution, gcp.WithLogOutput(false))
	if err != nil {
		return nil, err
	}
	return strings.Fields(result.Stdout), nil
}

func directDeps(ctx *gcp.Context) ([]string, error) {
	result, err := ctx.Exec([]string{"go", "list", "-e", "-f", `{{join .Imports "\n" }}`, "./..."}, gcp.WithUserAttribution, gcp.WithLogOutput(false))
	if err != nil {
		return nil, err
	}
	return strings.Fields(result.Stdout), nil
}
