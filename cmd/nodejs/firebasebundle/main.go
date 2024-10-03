// Copyright 2023 Google LLC
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

// Implements nodejs/firebasebundle buildpack.
// The output bundle buildpack sets up the output bundle for future steps
// It will do the following
// 1. Copy over static assets to the output bundle dir
// 2. Override run script with a new one to run the optimized build
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"gopkg.in/yaml.v2"
)

const (
	defaultPublicDir        = "public"
	firebaseOutputBundleDir = "FIREBASE_OUTPUT_BUNDLE_DIR"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// This buildpack handles some necessary setup for future app hosting processes,
	// it should always run for any app hosting initial build.
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	bundlePath := filepath.Join(ctx.ApplicationRoot(), ".apphosting", "bundle.yaml")
	bundleYaml, err := readBundleYaml(ctx, bundlePath)
	if err != nil {
		return err
	}

	outputBundleDir, ok := os.LookupEnv(firebaseOutputBundleDir)

	if !ok {
		return gcp.InternalErrorf("looking up output bundle env %s", firebaseOutputBundleDir)
	}

	workspacePublicDir := filepath.Join(ctx.ApplicationRoot(), defaultPublicDir)
	outputPublicDir := filepath.Join(outputBundleDir, defaultPublicDir)
	if bundleYaml == nil {
		ctx.Logf("bundle.yaml does not exist, assuming default configs")

		// if public folder exists assume that the code there should be in cdn
		ctx.Logf("No static assets declared, copying public directory (if it exists) to staticAssets by default")
		err := copyPublicDirToOutputBundleDir(outputPublicDir, workspacePublicDir, ctx)
		if err != nil {
			return err
		}
		err = generateDefaultBundleYaml(outputBundleDir, ctx)
		if err != nil {
			return err
		}

		return nil
	}

	ctx.Logf("Copying static assets.")
	err = fileutil.CopyFile(filepath.Join(outputBundleDir, "bundle.yaml"), bundlePath)
	if err != nil {
		return gcp.InternalErrorf("copying output bundle dir %s: %w", outputBundleDir, err)
	}
	err = copyPublicDirToOutputBundleDir(outputPublicDir, workspacePublicDir, ctx)
	if err != nil {
		return err
	}

	if bundleYaml.ServerConfig.RunCommand != "" {
		ctx.AddWebProcess(strings.Split(bundleYaml.ServerConfig.RunCommand, " "))
	}
	return nil
}

// bundleYaml represents the contents of a bundle.yaml file.
type bundleYaml struct {
	Version      string       `yaml:"version"`
	ServerConfig serverConfig `yaml:"serverConfig"`
	Metadata     metadata     `yaml:"metadata"`
}

type serverConfig struct {
	RunCommand           string         `yaml:"runCommand"`
	EnvironmentVariables []envVarConfig `yaml:"environmentVariables"`
	Concurrency          string         `yaml:"concurrency"`
	CPU                  string         `yaml:"cpu"`
	MemoryMiB            string         `yaml:"memory"`
	MinInstances         string         `yaml:"minInstances"`
	MaxInstances         string         `yaml:"maxInstances"`
}

type metadata struct {
	AdapterPackageName string `yaml:"name"`
	AdapterVersion     string `yaml:"path"`
	Framework          string `yaml:"framework"`
	FrameworkVersion   string `yaml:"frameworkVersion"`
}

type envVarConfig struct {
	Variable     string   `yaml:"variable"`
	Value        string   `yaml:"value"`
	Availability []string `yaml:"availability"`
}

func convertToMap(slice []string) map[string]bool {
	var newMap map[string]bool
	newMap = make(map[string]bool)
	for _, s := range slice {
		newMap[s] = true
	}
	return newMap
}

func readBundleYaml(ctx *gcp.Context, bundlePath string) (*bundleYaml, error) {
	bundleYamlExists, err := ctx.FileExists(bundlePath)
	if err != nil {
		return nil, err
	}
	if !bundleYamlExists {
		// return an empty struct if the file doesn't exist
		return nil, nil
	}
	rawBundleYaml, err := ctx.ReadFile(bundlePath)
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %w", bundlePath, err)
	}
	var bundleYaml bundleYaml
	if err := yaml.Unmarshal(rawBundleYaml, &bundleYaml); err != nil {
		return nil, gcp.UserErrorf("invalid %s: %w", bundlePath, err)
	}
	return &bundleYaml, nil
}

func generateDefaultBundleYaml(outputBundleDir string, ctx *gcp.Context) error {
	ctx.MkdirAll(outputBundleDir, 0744)
	f, err := ctx.CreateFile(filepath.Join(outputBundleDir, "bundle.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func copyPublicDirToOutputBundleDir(outputPublicDir string, workspacePublicDir string, ctx *gcp.Context) error {
	publicDirExists, err := ctx.FileExists(workspacePublicDir)
	if err != nil {
		return err
	}
	if !publicDirExists {
		return nil
	}
	err = ctx.MkdirAll(outputPublicDir, 0744)
	if err != nil {
		return err
	}
	if err := fileutil.MaybeCopyPathContents(outputPublicDir, workspacePublicDir, fileutil.AllPaths); err != nil {
		return err
	}
	return nil
}
