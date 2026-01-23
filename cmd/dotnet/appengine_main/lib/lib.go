// Copyright 2025 Google LLC
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

// Implements dotnet/appengine_main buildpack.
// The appengine_main buildpack handles the app.yaml `main` field when specified.
package lib

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsGAE() {
		return appengine.OptOutTargetPlatformNotGAE(), nil
	}
	if proj := os.Getenv(env.GAEMain); proj == "" {
		return gcp.OptOut("app.yaml main field is not defined, using default"), nil
	}

	if _, exists := os.LookupEnv(env.Buildable); exists {
		return gcp.OptOut(fmt.Sprintf("%s is set, ignoring app.yaml main field", env.Buildable)), nil
	}
	return gcp.OptIn("app.yaml found with the main field set"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("main_env", gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	l.BuildEnvironment.Override(env.Buildable, os.Getenv(env.GAEMain))
	return nil
}
