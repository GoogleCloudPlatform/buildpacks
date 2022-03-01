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
	"path/filepath"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	envFlexRe = regexp.MustCompile(`\s*env\s*:\s*(flex|flexible)\s*`)
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	path := os.Getenv(env.GaeApplicationYamlPath)

	if path == "" {
		return gcp.OptOut("Env var GAE_APPLICATION_YAML_PATH is not set, not a GAE Flex app."), nil
	}
	path = filepath.Join(ctx.ApplicationRoot(), path)
	pathExists, err := ctx.FileExists(path)
	if err != nil {
		return nil, err
	}
	if !pathExists {
		return gcp.OptOutFileNotFound(path), nil
	}
	content, err := ctx.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(envFlexRe.FindAllString(string(content), -1)) > 0 {
		return gcp.OptIn("env: flex found in the application yaml file."), nil
	}
	return gcp.OptOut("env: flex not found in the application yaml file."), nil
}

func buildFn(ctx *gcp.Context) error {
	return nil
}
