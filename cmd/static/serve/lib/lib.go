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

// Package lib provides the buildpack logic for serving static sites.
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
	// indexHTML is the name of the index.html file.
	indexHTML = "index.html"
	// nginxPathBaseImage is the path to the nginx configuration root in the static runtimes base image.
	nginxPathBaseImage = "/etc/nginx/"
	// nginxPathBuildpacks is the path to the nginx configuration root when installed by the nginx buildpack.
	nginxPathBuildpacks = "/layers/google.utils.nginx/nginx/conf"
)

// DetectFn checks for an index.html at the application root. Subdirectories
// such as build/ or dist/ are not scanned: the static files must be in the
// directory being deployed (for example, `--source ./dist`).
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Restrict this feature behind ALPHA release track.
	if !env.IsBetaSupported() {
		return gcp.OptOut("Static runtimes feature is supported only on ALPHA release track."), nil
	}

	info, err := os.Stat(filepath.Join(ctx.ApplicationRoot(), indexHTML))
	if err == nil && !info.IsDir() {
		return gcp.OptInFileFound(indexHTML), nil
	}

	return gcp.OptOut("index.html not found at the application root; subdirectories are not scanned, so deploy the directory that contains index.html (for example, --source ./dist)."), nil
}

// BuildFn generates or copies the nginx configuration and registers the web entrypoint.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("nginx_config", gcp.LaunchLayer, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	l.BuildEnvironment.Override(env.StaticServe, "true")

	rootPath := ctx.ApplicationRoot()
	ctx.Logf("Serving static files from the application root: %s", rootPath)

	nginxConfPath := filepath.Join(l.Path, static.NginxConfFile)
	ctx.Logf("Generating default SPA/SSG-friendly %s", static.NginxConfFile)

	nginxPath := nginxPathBuildpacks
	if env.IsStaticBaseImage() {
		nginxPath = nginxPathBaseImage
	}
	nginxMimeTypesPath := filepath.Join(nginxPath, "mime.types")

	params := static.NginxConfigParams{
		RootPath:      rootPath,
		MimeTypesPath: nginxMimeTypesPath,
	}
	if err := static.WriteNginxConfig(nginxConfPath, static.NginxConfigV1, params); err != nil {
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
