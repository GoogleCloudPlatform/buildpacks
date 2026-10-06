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

// Implements dotnet/publish buildpack.
// The publish buildpack runs dotnet publish.
package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, exists := os.LookupEnv(env.Buildable); exists {
		return gcp.OptInEnvSet(env.Buildable), nil
	}
	files, err := dotnet.ProjectFiles(ctx, ".")
	if err != nil {
		return nil, err
	}
	if len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}

	return gcp.OptOut(fmt.Sprintf("no project files found and %s not set", env.Buildable)), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	proj, err := dotnet.FindProjectFile(ctx)
	if err != nil {
		return fmt.Errorf("finding project: %w", err)
	}

	cap := ctx.Capability(dotnet.PublisherCapability)
	if cap != nil {
		publisher, ok := cap.(dotnet.Publisher)
		if !ok {
			return gcp.InternalErrorf("capability %q must implement dotnet.Publisher", dotnet.PublisherCapability)
		}
		return publisher.Publish(ctx, proj, os.Getenv(env.BuildArgs))
	}
	return dotnet.Publish(ctx, proj, os.Getenv(env.BuildArgs), true)
}
