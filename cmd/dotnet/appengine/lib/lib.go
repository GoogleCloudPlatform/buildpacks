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

// Implements dotnet/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package lib

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsGAE() {
		return appengine.OptInTargetPlatformGAE(), nil
	}
	return appengine.OptOutTargetPlatformNotGAE(), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	return appengine.Build(ctx, "dotnet", entrypoint)
}

func entrypoint(ctx *gcp.Context) (*appstart.Entrypoint, error) {
	ep := os.Getenv(env.Entrypoint)
	if ep == "" {
		return nil, gcp.UserErrorf("expected entrypoint from app.yaml or root project file, found nothing")
	}
	ctx.Logf("Using the entrypoint: %q", ep)
	return &appstart.Entrypoint{Type: appstart.EntrypointGenerated.String(), Command: ep}, nil
}
