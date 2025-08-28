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
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	dartLayer             = "dart"
	defaultVersion        = "2.16.0"
	flutterLayer          = "flutter"
	defaultFlutterVersion = "3.29.3"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("dart"); result != nil {
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

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	flutter, err := dart.IsFlutter(ctx.ApplicationRoot())
	if err == nil && flutter {
		return buildFlutterFn(ctx)
	}
	return buildDartFn(ctx)
}

func buildDartFn(ctx *gcp.Context) error {
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

	if runtime.IsCached(ctx, drl, version) {
		ctx.CacheHit(dartLayer)
		ctx.Logf("Runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(dartLayer)

	return runtime.InstallDartSDK(ctx, drl, version)
}

func buildFlutterFn(ctx *gcp.Context) error {
	version, archive, err := dart.DetectFlutterSDKArchive()
	if err != nil {
		return err
	}
	ctx.Logf("Using Flutter SDK version %s", version)

	// The Flutter SDK is only required at compile time. It is not included in the run image.
	drl, err := ctx.Layer(flutterLayer, gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", flutterLayer, err)
	}

	if runtime.IsCached(ctx, drl, version) {
		ctx.CacheHit(flutterLayer)
		ctx.Logf("Runtime cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(flutterLayer)

	return runtime.InstallFlutterSDK(ctx, drl, version, archive)
}
