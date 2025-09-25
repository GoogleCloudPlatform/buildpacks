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

// Implements nodejs/firebasebundle buildpack.
// The output bundle buildpack sets up the output bundle for future steps
// It will do the following
// 1. Copy over static assets to the output bundle dir
// 2. Override run script with a new one to run the optimized build
package main

import (
	"github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/firebasebundle/lib"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(lib.DetectFn, lib.BuildFn)
}
