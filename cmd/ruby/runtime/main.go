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

// Implements ruby/runtime buildpack.
// The runtime buildpack installs the Ruby runtime.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

// useRubyRuntime is used to enable the ruby/runtime buildpack
const useRubyRuntime = "GOOGLE_USE_EXPERIMENTAL_RUBY_RUNTIME"

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	isEnabled, err := env.IsPresentAndTrue(useRubyRuntime)
	if err != nil {
		ctx.Warnf("failed to parse %s: %v", useRubyRuntime, err)
	}

	if !isEnabled {
		return gcp.OptOutEnvNotSet(useRubyRuntime), nil
	}

	if result := runtime.CheckOverride(ctx, "ruby"); result != nil {
		return result, nil
	}

	if ctx.FileExists("Gemfile") {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	if ctx.FileExists("gems.rb") {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	if !ctx.HasAtLeastOne("*.rb") {
		return gcp.OptOut("no .rb files found"), nil
	}
	return gcp.OptIn("found .rb files"), nil

}

func buildFn(ctx *gcp.Context) error {
	version, err := ruby.DetectVersion(ctx)
	if err != nil {
		return fmt.Errorf("determining runtime version: %w", err)
	}
	rl := ctx.Layer("ruby", gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	return runtime.InstallTarball(ctx, runtime.Ruby, version, rl)
}
