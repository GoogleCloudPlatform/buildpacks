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
	// nginxPathBaseImage is the path to the nginx root in the static runtimes base image.
	nginxPathBaseImage = "/opt/nginx/"
	// nginxPathBuildpacks is the path to the nginx root when installed by the nginx buildpack.
	nginxPathBuildpacks = "/layers/google.utils.nginx/nginx"
)

// staticAssets is a list of potential static file/folder indicators, ordered by preference.
// Output directories take precedence over a raw index.html to ensure compiled distribution logic wins.
var staticAssets = []string{
	"build",
	"dist",
	"public",
	"_site",
	"site",
	indexHTML,
}

// DetectFn checks for standard static compiled directories or an index.html.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Restrict this feature behind ALPHA release track.
	if !env.IsAlphaSupported() {
		return gcp.OptOut("Static runtimes feature is supported only on ALPHA release track."), nil
	}

	for _, c := range staticAssets {
		fullPath := filepath.Join(ctx.ApplicationRoot(), c)
		info, err := os.Stat(fullPath)
		if err == nil {
			if c == indexHTML && !info.IsDir() {
				return gcp.OptInFileFound(c), nil
			}
			if c != indexHTML && info.IsDir() {
				return gcp.OptInFileFound(c), nil
			}
		}
	}

	return gcp.OptOut("No static asset folders or index.html found."), nil
}

// BuildFn generates or copies the nginx configuration and registers the web entrypoint.
func BuildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("nginx_config", gcp.LaunchLayer, gcp.BuildLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	l.BuildEnvironment.Override(env.StaticServe, "true")

	rootPath := ctx.ApplicationRoot()
	for _, c := range staticAssets {
		fullPath := filepath.Join(ctx.ApplicationRoot(), c)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if c == indexHTML && !info.IsDir() {
			ctx.Logf("Target static asset found: index.html at root.")
			break
		}
		if c != indexHTML && info.IsDir() {
			rootPath = fullPath
			ctx.Logf("Target static asset folder found: %s", c)
			break
		}
	}

	nginxConfPath := filepath.Join(l.Path, static.NginxConfFile)
	ctx.Logf("Generating default SPA/SSG-friendly %s", static.NginxConfFile)

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
