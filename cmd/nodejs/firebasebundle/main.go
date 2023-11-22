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
// 1. Copy over static assets to the build layer
// 2. Delete unnecessary files
// 3. Override run script with a new one to run the optimized build
package main

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"gopkg.in/yaml.v2"
)

const (
	defaultPublicDir = "public"
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

	staticAssetLayer, err := ctx.Layer("staticAssets", gcp.BuildLayer)
	if err != nil {
		return err
	}

	workspacePublicDir := filepath.Join(ctx.ApplicationRoot(), defaultPublicDir)
	layerPublicDir := filepath.Join(staticAssetLayer.Path, defaultPublicDir)
	if bundleYaml == nil {
		ctx.Logf("bundle.yaml does not exist, assuming default configs")

		// if public folder exists assume that the code there should be in cdn
		ctx.Logf("No static assets declared, copying public directory (if it exists) to staticAssets by default")
		err := ctx.MkdirAll(layerPublicDir, 0744)
		if err != nil {
			return err
		}
		fileutil.MaybeCopyPathContents(layerPublicDir, workspacePublicDir, fileutil.AllPaths)

		return nil
	}

	ctx.Logf("Copying static assets.")
	fileutil.CopyFile(filepath.Join(staticAssetLayer.Path, "bundle.yaml"), bundlePath)

	if bundleYaml.StaticAssets == nil {
		// copy public folder by default if there are no static assets declared
		ctx.Logf("No static assets declared, copying public directory (if it exists) to staticAssets by default")
		err := ctx.MkdirAll(layerPublicDir, 0744)
		if err != nil {
			return err
		}
		fileutil.MaybeCopyPathContents(layerPublicDir, workspacePublicDir, fileutil.AllPaths)
	} else {
		for _, staticAsset := range bundleYaml.StaticAssets {
			ctx.MkdirAll(filepath.Join(staticAssetLayer.Path, staticAsset), 0744)
			err := fileutil.MaybeCopyPathContents(filepath.Join(staticAssetLayer.Path, staticAsset), filepath.Join(ctx.ApplicationRoot(), staticAsset), fileutil.AllPaths)
			if err != nil {
				ctx.Logf("%s dir not detected", staticAsset)
			}
		}
	}

	ctx.Logf("Deleting unneeded dirs.")
	if bundleYaml.NeededDirs == nil {
		ctx.Logf("No directories declared, keeping all by default")
	} else {
		files, err := ctx.ReadDir(ctx.ApplicationRoot())
		if err != nil {
			return err
		}
		for _, file := range files {
			if !slices.Contains(bundleYaml.NeededDirs, file.Name()) {
				err := ctx.RemoveAll(filepath.Join(ctx.ApplicationRoot(), file.Name()))
				if err != nil {
					return err
				}
			}
		}
	}

	ctx.Logf("Configuring run command entry point")
	if bundleYaml.RunCommand != "" {
		ctx.AddWebProcess(strings.Split(bundleYaml.RunCommand, " "))
	}
	return nil
}

// BundleYaml represents the contents of a bundle.yaml file.
type bundleYaml struct {
	RunCommand   string   `yaml:"runCommand"`
	NeededDirs   []string `yaml:"neededDirs"`
	StaticAssets []string `yaml:"staticAssets"`
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
