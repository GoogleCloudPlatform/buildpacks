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

// Binary dotnet/runtime buildpack detects .NET applications
// and install the corresponding version of .NET runtime.
package main

import (
	lib "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/sdk/lib"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(lib.DetectFn, lib.BuildFn)
}
