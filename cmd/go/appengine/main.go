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
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
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

	if !supportsApis && appEngineInDeps(directDeps(ctx)) {
		// TODO(b/179431689) Change to error.
		ctx.Warnf(appengine.DepWarning)
		return nil
	}

	usingAppEngine := appEngineInDeps(allDeps(ctx))
	if supportsApis && !usingAppEngine {
		ctx.Warnf(appengine.UnusedAPIWarning)
	}

	if !supportsApis && usingAppEngine {
		ctx.Warnf(appengine.IndirectDepWarning)
	}

	return nil
}

func entrypoint(ctx *gcp.Context) (*appengine.Entrypoint, error) {
	ctx.Logf("No user entrypoint specified. Using the generated entrypoint %q", golang.OutBin)
	return &appengine.Entrypoint{Type: appengine.EntrypointGenerated.String(), Command: golang.OutBin}, nil
}

func appEngineInDeps(deps []string) bool {
	for _, s := range deps {
		if strings.HasPrefix(s, "google.golang.org/appengine") {
			return true
		}
	}
	return false
}

func allDeps(ctx *gcp.Context) []string {
	result := ctx.Exec([]string{"go", "list", "-f", `{{join .Deps "\n"}}`, "./..."}, gcp.WithUserAttribution)

	return strings.Fields(result.Stdout)
}

func directDeps(ctx *gcp.Context) []string {
	result := ctx.Exec([]string{"go", "list", "-f", `{{join .Imports "\n" }}`, "./..."}, gcp.WithUserAttribution)

	return strings.Fields(result.Stdout)
}
