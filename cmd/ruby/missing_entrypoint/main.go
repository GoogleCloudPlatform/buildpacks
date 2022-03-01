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

// Implements ruby/missing-entrypoint buildpack.
// This buildpack's goal is to display a clear error message when
// no entrypoint is defined on a Ruby application.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride(ctx, "ruby"); result != nil {
		return result, nil
	}

	atLeastOne, err := ctx.HasAtLeastOne("*.rb")
	if err != nil {
		return nil, fmt.Errorf("finding *.rb files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .rb files found"), nil
	}
	return gcp.OptIn("found .rb files"), nil
}

func buildFn(ctx *gcp.Context) error {
	return fmt.Errorf("for Ruby, an entrypoint must be manually set, either with %q env var or by creating a %q file", env.Entrypoint, "Procfile")
}
