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

// Implements go/appengine_gopath buildpack.
// The appengine_gopath buildpack sets $GOPATH and moves all gopath dependencies from _gopath/src/* to $GOPATH/src/*. The _gopath directory is created by go-app-stager during deployment.
// It then checks for _gopath/main-package-path which exists if the user's main package was originally on $GOPATH/src locally.
// If this file exists, the buildpack moves the main package to $GOPATH/src and sets the path to build $GOPATH/src/<path-to-main-package> where <path-to-main-package> is read from _gopath/main-package-path.
// If this file doesn't exist, the buildpack sets the path to build to "./..." and removes the _gopath directory because the build will fail if there's more than one go package in application root.
package main

import (
	lib "github.com/GoogleCloudPlatform/buildpacks/cmd/go/appengine_gopath/lib"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(lib.DetectFn, lib.BuildFn)
}
