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

// Implements config/entrypoint buildpack.
// The entrypoint buildpack sets the image entrypoint based on an environment variable or Procfile.
package main

import (
	"os"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	webRegexp = regexp.MustCompile(`(?m)^web:\s*(.+)$`)
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if os.Getenv(env.Entrypoint) == "" && !ctx.FileExists("Procfile") {
		ctx.OptOut("%s not set and Procfile not found", env.Entrypoint)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	entrypoint := os.Getenv(env.Entrypoint)
	if entrypoint != "" {
		ctx.Logf("Using entrypoint from %s: %s", env.Entrypoint, entrypoint)
	} else {
		b := ctx.ReadFile("Procfile")
		var err error
		entrypoint, err = procfileWebProcess(string(b))
		if err != nil {
			return err
		}
		ctx.Logf("Using entrypoint from Procfile: %s", entrypoint)
	}
	// Use /bin/bash because lifecycle/launcher will assume the whole command is a single executable.
	ctx.AddWebProcess([]string{"/bin/bash", "-c", entrypoint})
	return nil
}

func procfileWebProcess(content string) (string, error) {
	matches := webRegexp.FindStringSubmatch(content)
	if len(matches) != 2 {
		return "", gcp.UserErrorf("could not find web process in Procfile: %v", matches)
	}
	return matches[1], nil
}
