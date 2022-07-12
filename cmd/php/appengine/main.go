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

// Implements php/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package main

import (
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsGAE() {
		return gcp.OptInEnvSet(env.XGoogleTargetPlatform), nil
	}

	return gcp.OptOutEnvNotSet(env.XGoogleTargetPlatform), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := validateAppEngineAPIs(ctx); err != nil {
		return err
	}
	return appengine.Build(ctx, "php", nil)
}

func validateAppEngineAPIs(ctx *gcp.Context) error {
	composerExists, err := ctx.FileExists("composer.json")
	if err != nil {
		return err
	}
	if !composerExists {
		return nil
	}

	supportsApis, err := php.SupportsAppEngineApis(ctx)
	if err != nil {
		return err
	}

	dirDeps, err := directDeps(ctx)
	if err != nil {
		return err
	}

	if !supportsApis && appEngineInDeps(dirDeps) {
		ctx.Warnf("There is a direct dependency on App Engine APIs, but they are not enabled in app.yaml (set the app_engine_apis property)")
		return nil
	}

	aDeps, err := allDeps(ctx)
	if err != nil {
		return err
	}

	usingAppEngine := appEngineInDeps(aDeps)
	if supportsApis && !usingAppEngine {
		ctx.Warnf("App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app's yaml config file.")
		return nil
	}

	if !supportsApis && usingAppEngine {
		ctx.Warnf("There is an indirect dependency on App Engine APIs, but they are not enabled in app.yaml. You may see runtime errors trying to access these APIs. Set the app_engine_apis property.")
	}

	return nil
}

func appEngineInDeps(deps []string) bool {
	for _, s := range deps {
		if strings.HasPrefix(s, "google/appengine-php-sdk") {
			return true
		}
	}
	return false
}

func allDeps(ctx *gcp.Context) ([]string, error) {
	result, err := ctx.ExecWithErr([]string{"composer", "show", "-N"}, gcp.WithUserAttribution)
	if err != nil {
		return nil, err
	}
	return strings.Fields(result.Stdout), nil
}

func directDeps(ctx *gcp.Context) ([]string, error) {
	result, err := ctx.ExecWithErr([]string{"composer", "show", "--direct", "-N"}, gcp.WithUserAttribution)
	if err != nil {
		return nil, err
	}
	return strings.Fields(result.Stdout), nil
}
