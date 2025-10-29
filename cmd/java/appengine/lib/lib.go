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
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	java25Runtime = "java25"
)

var (
	supportedJettyBuildTimeVersions = []string{
		java25Runtime,
	}
)

// EEConfig contains the files to delete from the Jetty distribution for a given runtime.
type EEConfig struct {
	EE8Deletions     []string
	EE10Deletions    []string
	EE11Deletions    []string
	defaultEEVersion string
}

var jettyFilesToDelete = map[string]EEConfig{
	java25Runtime: {
		// This is based on the behavior defined in ClassPathUtils.
		// See for reference:
		// http://google3/third_party/java_src/appengine_standard/runtime/util/src/main/java/com/google/apphosting/runtime/ClassPathUtils.java;l=111
		EE8Deletions:     []string{"runtime-shared-jetty121-ee11.jar"},
		EE10Deletions:    []string{},
		EE11Deletions:    []string{"runtime-shared-jetty121-ee8.jar"},
		defaultEEVersion: "EE11",
	},
}

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
		return addJettyAtBuildTime(ctx, appEngineWebXMLApp)
	}

	return "", nil
}

func addJettyAtBuildTime(ctx *gcp.Context, appEngineWebXMLApp *java.AppEngineWebXMLApp) (string, error) {
	jettyLayer, err := ctx.Layer("java_runtime", gcp.LaunchLayer)
	repoPath := appEngineWebXMLApp.Runtime
	if err != nil {
		return "", fmt.Errorf("creating layer: %w", err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Jetty, "", jettyLayer)
	if err != nil {
		return "", fmt.Errorf("Error installing jetty artifacts: %w", err)
	}
	ctx.Logf("Successfully installed Jetty for %s at build time from AR.", repoPath)

	err = handleRuntimeJettyFiles(ctx, appEngineWebXMLApp, jettyLayer.Path)
	if err != nil {
		return "", err
	}
	return jettyLayer.Path, nil
}

// handleRuntimeJettyFiles tailors the installed Jetty distribution based on runtime configuration.
func handleRuntimeJettyFiles(ctx *gcp.Context, appEngineWebXMLApp *java.AppEngineWebXMLApp, jettyRoot string) error {
	config, exists := jettyFilesToDelete[appEngineWebXMLApp.Runtime]
	if !exists {
		return nil
	}

	eeVersion, err := extractEEVersion(appEngineWebXMLApp, config.defaultEEVersion, ctx)
	if err != nil {
		return err
	}

	var fileNamesToDelete []string
	switch eeVersion {
	case "EE8":
		fileNamesToDelete = config.EE8Deletions
	case "EE11":
		fileNamesToDelete = config.EE11Deletions
	}

	if len(fileNamesToDelete) == 0 {
		return nil
	}

	err = filepath.Walk(jettyRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileName := info.Name()
		if slices.Contains(fileNamesToDelete, fileName) || !strings.HasSuffix(fileName, ".jar") {
			if rmErr := os.Remove(path); rmErr != nil {
				ctx.Logf("Warning: Failed to delete file %s: %v", path, rmErr)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func extractEEVersion(appEngineWebXMLApp *java.AppEngineWebXMLApp, defaultEEVersion string, ctx *gcp.Context) (string, error) {
	var trueCount int = 0
	var eeVersion string
	var err error = nil
	// First return an error when more than one appengine.use.* properties are true.
	// If that is not the case, return an error when appengine.use.EE10 is true.
	// Otherwise, return the value of the last true property.
	for _, prop := range appEngineWebXMLApp.SystemProperties {
		// case insensitive comparison for value of the system property
		// "appengine.use.EE11" can be "true" or "True", for e.g.
		if prop.Name == "appengine.use.EE11" && strings.ToLower(prop.Value) == "true" {
			trueCount++
			eeVersion = "EE11"
		} else if prop.Name == "appengine.use.EE8" && strings.ToLower(prop.Value) == "true" {
			trueCount++
			eeVersion = "EE8"
		} else if prop.Name == "appengine.use.EE10" && strings.ToLower(prop.Value) == "true" {
			trueCount++
			eeVersion = "EE10"
			err = fmt.Errorf("appengine.use.EE10 is not supported in Jetty121")
		}
		if trueCount > 1 {
			return "", fmt.Errorf("only one of appengine.use.EE8, appengine.use.EE10, or appengine.use.EE11 can be true")
		}
	}
	if trueCount == 0 {
		eeVersion = defaultEEVersion
		ctx.Logf("No appengine.use.* property found in appengine-web.xml, using default EE version: %s", eeVersion)
	}
	return eeVersion, err
}
