// Copyright 2023 Google LLC
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

// Implements nodejs/firebasenextjs buildpack.
// The nodejs/firebasenextjs buildpack does some prep work for nextjs and overwrites the build script.
package main

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// TODO (b/311402770)
	// In monorepo scenarios, we'll probably need to support environment variable that can be used to
	// know where the application directory is located within the repository.
	nextConfigExists, err := ctx.FileExists("next.config.js")
	if err != nil {
		return nil, err
	}
	if nextConfigExists {
		return gcp.OptInFileFound("next.config.js"), nil
	}

	nextConfigModuleExists, err := ctx.FileExists("next.config.mjs")
	if err != nil {
		return nil, err
	}
	if nextConfigModuleExists {
		return gcp.OptInFileFound("next.config.mjs"), nil
	}
	return gcp.OptOut("nextjs config not found"), nil
}

func buildFn(ctx *gcp.Context) error {
	return nil
}
