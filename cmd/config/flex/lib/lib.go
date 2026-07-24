// Copyright 2025 Google LLC
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
package lib

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

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsFlex() {
		return gcp.OptInEnvSet(env.XGoogleTargetPlatform), nil
	}

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

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	layer, err := ctx.Layer("flex", gcp.BuildLayer)
	if err != nil {
		return err
	}
	layer.BuildEnvironment.Default(env.FlexEnv, true)

	return nil
}
