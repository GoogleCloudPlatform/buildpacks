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

// Implements /bin/build for nodejs/functions-framework buildpack.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/blang/semver"
)

var (
	minVer = semver.MustParse("3.4.0")
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); !ok {
		ctx.OptOut("%s not set.", env.FunctionTarget)
	}

	if !ctx.FileExists("package.json") {
		ctx.OptOut("package.json not found.")
	}

	pjs, err := nodejs.ReadPackageJSON(ctx.ApplicationRoot())
	if err != nil {
		return gcp.Errorf(gcp.StatusInvalidArgument, "reading package.json in %q: %v", ctx.ApplicationRoot(), err)
	}

	if _, ok := pjs.Dependencies["firebase-functions"]; !ok {
		ctx.OptOut("firebase-functions not found in package.json.")
	}

	return nil
}

func buildFn(ctx *gcp.Context) error {
	fbDir := filepath.Join("node_modules", "firebase-functions")
	fbVer := ""

	// Determine the installed version of the firebase-functions.
	if !ctx.FileExists(fbDir, "package.json") {
		return fmt.Errorf("could not find package.json in %s", fbDir)
	}

	pjs, err := nodejs.ReadPackageJSON(fbDir)
	if err != nil {
		return gcp.Errorf(gcp.StatusInvalidArgument, "reading package.json in %q: %v", fbDir, err)
	}
	fbVer = pjs.Version
	ctx.Logf("Using firebase-functions v%s.", fbVer)

	// Fail if firebase-functions less than v3.4.0.
	if v, err := semver.Parse(fbVer); err != nil {
		return gcp.UserErrorf("could not parse firebase-functions version string %q: %v", fbVer, err)
	} else if v.LT(minVer) {
		return gcp.UserErrorf("firebase-functions v%s is not supported. Please update to firebase-functions v3.4.0 or above.", fbVer)
	}

	return nil
}
