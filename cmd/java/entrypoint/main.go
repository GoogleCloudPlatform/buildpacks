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

// Implements the java/entrypoint buildpack.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	executable, err := java.ExecutableJar(ctx)
	if err != nil {
		return fmt.Errorf("finding executable jar: %w", err)
	}

	command := []string{"java", "-jar", executable}

	// Configure the entrypoint and metadata for dev mode.
	if devmode.Enabled(ctx) {
		devmode.AddSyncMetadata(ctx, devmode.JavaSyncRules)
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
