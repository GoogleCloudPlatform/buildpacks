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

// Implements flutter/sdk buildpack.
// The sdk buildpack installs the Fart SDK.
package main

import (
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dart"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	flutterLayer   = "flutter"
	defaultVersion = "3.29.3"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("flutter"); result != nil {
		return result, nil
	}
	pubspecExists, err := ctx.FileExists("pubspec.yaml")
	if err != nil {
		return nil, err
	}
	if pubspecExists {
		flutter, err := dart.IsFlutter(ctx.ApplicationRoot())
		if err != nil {
			return nil, err
		}
		if !flutter {
			return gcp.OptOut("pubspec.yaml does not include flutter dependency"), nil
		}
		return gcp.OptInFileFound("pubspec.yaml"), nil
	}

	return gcp.OptOut("no pubspec.yaml file found"), nil
}

func buildFn(ctx *gcp.Context) error {
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
