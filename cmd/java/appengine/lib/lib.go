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

// Implements java/appengine buildpack.
// The appengine buildpack sets the image entrypoint.
package lib

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

var (
	supportedJettyBuildTimeVersions = []string{
		"java25",
	}
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsGAE() {
		return appengine.OptInTargetPlatformGAE(), nil
	}
	return appengine.OptOutTargetPlatformNotGAE(), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	return appengine.Build(ctx, "java", entrypoint)
}

func entrypoint(ctx *gcp.Context) (*appstart.Entrypoint, error) {
	webXMLExists, err := ctx.FileExists("WEB-INF", "appengine-web.xml")
	if err != nil {
		return nil, err
	}
	if webXMLExists {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.JavaGAEWebXMLConfigUsageCounterID).Increment(1)
		jettyLayer, err := processAppEngineWebXML(ctx)
		if err != nil {
			return nil, fmt.Errorf("Error processing appengine-web.xml: %w", err)
		}
		if jettyLayer != "" {
			ctx.Logf("WAR packaging detected and injecting the embedded web-server dependencies at build time")
			return &appstart.Entrypoint{
				Type:    appstart.EntrypointGenerated.String(),
				Command: "serve WEB-INF/appengine-web.xml " + jettyLayer,
			}, nil
		}
		return &appstart.Entrypoint{
			Type:    appstart.EntrypointGenerated.String(),
			Command: "serve WEB-INF/appengine-web.xml",
		}, nil
	}

	executable, err := java.ExecutableJar(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding executable jar: %w", err)
	}

	return &appstart.Entrypoint{
		Type:    appstart.EntrypointGenerated.String(),
		Command: "serve " + executable,
	}, nil
}

func processAppEngineWebXML(ctx *gcp.Context) (string, error) {
	fullPath := filepath.Join(ctx.ApplicationRoot(), "WEB-INF/appengine-web.xml")
	appEngineWebXML, err := ctx.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("Error reading appengine-web.xml: %v", err)
	}

	appEngineWebXMLApp, err := java.ParseAppEngineWebXML(appEngineWebXML)
	if err != nil {
		return "", fmt.Errorf("Error parsing appengine-web.xml: %w", err)
	}

	if appEngineWebXMLApp.SessionsEnabled {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.JavaGAESessionsEnabledCounterID).Increment(1)
	}

	if slices.Contains(supportedJettyBuildTimeVersions, appEngineWebXMLApp.Runtime) {
		return addJettyAtBuildTime(ctx, appEngineWebXMLApp.Runtime)
	}

	return "", nil
}

func addJettyAtBuildTime(ctx *gcp.Context, repoPath string) (string, error) {
	jettyLayer, err := ctx.Layer("java_runtime", gcp.LaunchLayer)
	if err != nil {
		return "", fmt.Errorf("creating layer: %w", err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Jetty, "", jettyLayer)
	if err != nil {
		return "", fmt.Errorf("Error installing jetty artifacts: %w", err)
	}
	ctx.Logf("Successfully installed Jetty for %s at build time from AR.", repoPath)
	return jettyLayer.Path, nil
}
