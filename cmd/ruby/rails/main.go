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

// Implements ruby/rails buildpack.
// The rails buildpack precompiles assets using Rails.
package main

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
)

const (
	yarnLayer = "yarn"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	railsExists, err := ctx.FileExists("bin", "rails")
	if err != nil {
		return nil, err
	}
	if !railsExists {
		return gcp.OptOutFileNotFound("bin/rails"), nil
	}
	needsPrecompile, err := ruby.NeedsRailsAssetPrecompile(ctx)
	if err != nil {
		return nil, err
	}
	if !needsPrecompile {
		return gcp.OptOut("Rails assets do not need precompilation"), nil
	}
	return gcp.OptIn("found Rails assets to precompile"), nil
}

func buildFn(ctx *gcp.Context) error {
	ctx.Logf("Running Rails asset precompilation")

	// Install Yarn as it is needed for asset precompilation.
	if err := installYarn(ctx); err != nil {
		return fmt.Errorf("installing Yarn: %w", err)
	}

	// It is common practise in Ruby asset precompilation to ignore non-zero exit codes.
	result, err := ctx.Exec([]string{"bundle", "exec", "ruby", "bin/rails", "assets:precompile"},
		gcp.WithEnv("RAILS_ENV=production", "MALLOC_ARENA_MAX=2", "RAILS_LOG_TO_STDOUT=true", "LANG=C.utf8"), gcp.WithUserAttribution)
	if err != nil && result != nil && result.ExitCode != 0 {
		ctx.Logf("WARNING: Asset precompilation returned non-zero exit code %d. Ignoring.", result.ExitCode)
		return nil
	}
	if err != nil && result != nil {
		return gcp.UserErrorf(result.Combined)
	}
	if err != nil {
		return gcp.InternalErrorf("asset precompilation failed: %v", err)
	}

	return nil
}

func installYarn(ctx *gcp.Context) error {
	yrl, err := ctx.Layer(yarnLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", yarnLayer, err)
	}
	return nodejs.InstallYarnLayer(ctx, yrl)
}
