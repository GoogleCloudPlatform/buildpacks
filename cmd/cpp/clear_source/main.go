// Copyright 2021 Google LLC
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

// Implements cpp/clear_source buildpack.
// The clear_source buildpack deletes source files after building the application.
package main

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/clearsource"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result, err := clearsource.DetectFn(ctx); result != nil || err != nil {
		return result, err
	}
	return gcp.OptInEnvSet(env.ClearSource), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	return clearsource.BuildFn(ctx, nil)
}
