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

// Implements dart/sdk buildpack.
// The sdk buildpack installs the Dart SDK.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dart"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	dartLayer      = "dart"
	defaultVersion = "2.16.0"
	dartEnabledEnv = "GOOGLE_DART_ENABLED"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !isDartEnabled() {
		return gcp.OptOutEnvNotSet(dartEnabledEnv), nil
	}
	if result := runtime.CheckOverride(ctx, "dart"); result != nil {
		return result, nil
	}
	pubspecExists, err := ctx.FileExists("pubspec.yaml")
	if err != nil {
		return nil, err
	}
	if pubspecExists {
		return gcp.OptInFileFound("pubspec.yaml"), nil
	}
	dartFiles, err := ctx.Glob("*.dart")
	if err != nil {
		return nil, fmt.Errorf("finding .dart files: %w", err)
	}
	if len(dartFiles) > 0 {
		return gcp.OptIn("found .dart files"), nil
	}

	return gcp.OptOut("neither pubspec.yaml nor any .dart files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	version, err := dart.DetectSDKVersion()
	if err != nil {
		return err
	}
	ctx.Logf("Using Dart SDK version %s", version)

	// The Dart SDK is only required at compile time. It is not included in the run image.
	drl, err := ctx.Layer(dartLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", dartLayer, err)
	}
	ctx.AddBOMEntry(libcnb.BOMEntry{
		Name:     dartLayer,
		Metadata: map[string]interface{}{"version": version},
		Build:    true,
	})

	if runtime.IsCached(ctx, drl, version) {
		ctx.CacheHit(dartLayer)
		ctx.Logf("Runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(dartLayer)

	return runtime.InstallDartSDK(ctx, drl, version)
}

// isDartEnabled returns true if we should enable the experimental Dart buildpacks.
func isDartEnabled() bool {
	res, err := env.IsPresentAndTrue(dartEnabledEnv)
	return err == nil && res
}
