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

// Implements dart/pub buildpack.
// The pub buildpack installs application dependencies using the pub package manager.
package main

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	pubLayer = "pub"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.FileExists("pubspec.yaml") {
		return gcp.OptOutFileNotFound("pubspec.yaml"), nil
	}
	return gcp.OptInFileFound("pubspec.yaml"), nil
}

func buildFn(ctx *gcp.Context) error {
	ml := ctx.Layer(pubLayer, gcp.BuildLayer, gcp.CacheLayer)
	ml.BuildEnvironment.Override("PUB_CACHE", ml.Path)
	ctx.Exec([]string{"dart", "pub", "get"}, gcp.WithUserAttribution)
	return nil
}
