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

// Implements ruby entrypoint buildpack for flex.
// Ruby Rails and Bundle uses localhost(127.0.0.1) as the default host which prevents requests
// external to the docker container from accessing the process. We need to explicitly bind it to
// "0.0.0.0" inorder to access it from outside the container.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/ruby"
)

const flexHost = "0.0.0.0"
const railsCommand = "rails "

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !env.IsFlex() {
		return gcp.OptOut("Not a GAE Flex app."), nil
	}
	// Detection for GCP builds follows
	if os.Getenv(env.Entrypoint) != "" {
		return gcp.OptInEnvSet(env.Entrypoint), nil
	}
	entrypoint, err := appyaml.EntrypointIfExists(ctx.ApplicationRoot())
	if err != nil {
		return nil, fmt.Errorf("Error finding entrypoint in app.yaml if set. %w", err)
	}
	if entrypoint != "" {
		ctx.Logf("Using entrypoint from app.yaml.")
		return gcp.OptIn("Found the app.yaml file specified by GAE_APPLICATION_YAML_PATH."), nil
	}
	return gcp.OptOut(env.Entrypoint + " not set, no valid entrypoint in app.yaml"), nil
}

func buildFn(ctx *gcp.Context) error {
	entrypoint := getEntrypoint(ctx)
	if strings.Contains(entrypoint, railsCommand) {
		// -b will "bind" the server to 0.0.0.0 instead of localhost.
		// https://guides.rubyonrails.org/command_line.html
		entrypoint = fmt.Sprintf("%s -b %s", entrypoint, flexHost)
	} else {
		entrypoint = fmt.Sprintf("%s -o %s", entrypoint, flexHost)
	}
	ctx.Logf("Using entrypoint %s", entrypoint)
	ctx.AddProcess(gcp.WebProcess, []string{entrypoint}, gcp.AsDefaultProcess())
	return nil
}

func getEntrypoint(ctx *gcp.Context) string {
	if entrypoint := os.Getenv(env.Entrypoint); entrypoint != "" {
		return entrypoint
	}

	entrypoint, err := appyaml.EntrypointIfExists(ctx.ApplicationRoot())
	if err != nil {
		ctx.Logf("app.yaml env var set but the specified app.yaml file doesn't exist.")
		return ""
	}

	if entrypoint != "" {
		return entrypoint
	}
	ep, err := ruby.InferEntrypoint(ctx, ctx.ApplicationRoot())
	if err != nil {
		ctx.Logf(err.Error())
		return ""
	}
	return ep
}
