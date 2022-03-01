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

// Implements utils/nginx buildpack.
// The nginx buildpack installs the nginx web server.
package main

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	// nginxVerConstraint is used to control updating to a new major version with any potential breaking change.
	// Update this to allow a new major version.
	nginxVerConstraint = "^1.21.6"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Always opt in.
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("nginx", gcp.CacheLayer, gcp.LaunchLayer)
	_, err := runtime.InstallTarballIfNotCached(ctx, runtime.Nginx, nginxVerConstraint, l)
	return err
}
