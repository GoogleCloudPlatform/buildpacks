// Copyright 2024 Google LLC
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

// Package util provides utility functions to build applications using the Firebase App Hosting builder.
package util

import (
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// ApplicationDirectory looks up the path to the application directory from the environment. Returns
// the application root by default.
func ApplicationDirectory(ctx *gcp.Context) string {
	appDir := ctx.ApplicationRoot()
	if appDirEnv, exists := os.LookupEnv(env.Buildable); exists {
		appDir = filepath.Join(ctx.ApplicationRoot(), appDirEnv)
	}
	return appDir
}
