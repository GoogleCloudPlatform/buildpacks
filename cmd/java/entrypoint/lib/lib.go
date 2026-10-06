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

// Implements the java/entrypoint buildpack.
package lib

import (
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	if entrypoint := getEntrypoint(ctx); env.IsFlex() && entrypoint != "" {
		ctx.Setenv(env.Entrypoint, entrypoint)
		appengine.Build(ctx, "java", nil)
		return nil
	}

	executable, err := java.ExecutableJar(ctx)
	if err != nil {
		return fmt.Errorf("finding executable jar: %w", err)
	}

	command := []string{"java", "-jar", executable}

	// Configure the entrypoint and metadata for dev mode.
	if devmode.Enabled(ctx) {
		if err := devmode.AddFileWatcherProcess(ctx, devmode.Config{
			BuildCmd: []string{".devmode_rebuild.sh"},
			RunCmd:   command,
			Ext:      devmode.JavaWatchedExtensions,
		}); err != nil {
			return fmt.Errorf("adding devmode file watcher: %w", err)
		}

		return nil
	}

	// Configure the entrypoint for production.
	ctx.AddWebProcess(command)
	return nil
}

func getEntrypoint(ctx *gcp.Context) string {
	if entrypoint := os.Getenv(env.Entrypoint); entrypoint != "" {
		return entrypoint
	}
	entrypoint, _ := appyaml.EntrypointIfExists(ctx.ApplicationRoot())
	return entrypoint
}
