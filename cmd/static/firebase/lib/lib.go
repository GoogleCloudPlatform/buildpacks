// Copyright 2026 Google LLC
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

// Package lib provides the buildpack logic for serving static sites configured with firebase.json.
package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/static"
)

const (
	// nginxPathBaseImage is the path to the nginx root in the static runtimes base image.
	nginxPathBaseImage = "/opt/nginx/"
	// nginxPathBuildpacks is the path to the nginx root when installed by the nginx buildpack.
	nginxPathBuildpacks = "/layers/google.utils.nginx/nginx"
)

// DetectFn checks for firebase.json with a valid public directory.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Restrict this feature behind ALPHA release track.
	if !env.IsAlphaSupported() {
		return gcp.OptOut("Static runtimes feature is supported only on ALPHA release track."), nil
	}

	firebaseConfigPath := filepath.Join(ctx.ApplicationRoot(), "firebase.json")
	if configs, err := static.ParseFirebaseConfig(firebaseConfigPath); err == nil && len(configs) > 0 {
		// Default to the first config in the array. App Hosting currently only supports deploying
		// a single target per container, and we do not yet have plumbing to receive a specific
		// target identifier (like `firebase deploy --only hosting:target`) from the user.
		if publicDir := configs[0].Public; publicDir != "" {
			fullPath := filepath.Join(ctx.ApplicationRoot(), publicDir)
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				return gcp.OptInFileFound("firebase.json (public: " + publicDir + ")"), nil
			}
		}
	}

	return gcp.OptOut("No valid firebase.json public asset folder found."), nil
}

// BuildFn generates the nginx configuration for a firebase.json application and registers the web entrypoint.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("nginx_config", gcp.LaunchLayer, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	l.BuildEnvironment.Override(env.StaticServe, "true")

	rootPath := ctx.ApplicationRoot()
	var fbConfig *static.HostingConfig

	firebaseConfigPath := filepath.Join(ctx.ApplicationRoot(), "firebase.json")
	if info, err := os.Stat(firebaseConfigPath); err == nil && !info.IsDir() {
		ctx.Logf("Found firebase.json at application root. Parsing...")
		configs, err := static.ParseFirebaseConfig(firebaseConfigPath)
		if err != nil {
			return fmt.Errorf("parsing firebase.json: %w", err)
		}
		if len(configs) > 0 {
			fbConfig = &configs[0]
			ctx.Logf("Successfully parsed firebase.json. Applied %d custom header rules.", len(fbConfig.Headers))
			if fbConfig.Public != "" {
				fullPath := filepath.Join(ctx.ApplicationRoot(), fbConfig.Public)
				if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
					rootPath = fullPath
					ctx.Logf("Target static asset folder found via firebase.json: %s", fbConfig.Public)
				}
			}
		}
	}

	nginxConfPath := filepath.Join(l.Path, static.NginxConfFile)
	ctx.Logf("Generating default SPA/SSG-friendly %s for firebase.json", static.NginxConfFile)

	nginxPath := nginxPathBuildpacks
	if env.IsStaticBaseImage() {
		nginxPath = nginxPathBaseImage
	}
	nginxMimeTypesPath := filepath.Join(nginxPath, "conf/mime.types")

	params := static.NginxConfigParams{
		RootPath:      rootPath,
		MimeTypesPath: nginxMimeTypesPath,
	}
	if err := static.WriteNginxConfig(nginxConfPath, params); err != nil {
		return fmt.Errorf("writing %s: %w", static.NginxConfFile, err)
	}

	// Setup Entrypoint
	ctx.AddProcess(gcp.WebProcess, []string{
		"nginx",
		// TODO(b/512020384) - remove explicit path once we have dedicated nginx tarball for static case.
		"-p", nginxPath,
		"-c", nginxConfPath,
		"-g", "daemon off;",
	}, gcp.AsDefaultProcess(), gcp.AsDirectProcess())

	return nil
}
