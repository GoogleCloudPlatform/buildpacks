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

// Implements php/cloudfunctions buildpack.
// The cloudfunctions buildpack sets the image entrypoint.
package main

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	// routerScript is the path to the functions framework invoker script.
	routerScript = "vendor/bin/router.php"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Always opt in.
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	return cloudfunctions.Build(ctx, "php", entrypoint)
}

func entrypoint(*gcp.Context) (*appstart.Entrypoint, error) {
	return &appstart.Entrypoint{
		Type:    appstart.EntrypointGenerated.String(),
		Command: "serve " + routerScript,
	}, nil
}
