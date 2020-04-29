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

// Implements go/gomod buildpack.
// The gomod buildpack downloads modules specified in go.mod.
package main

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/buildpack/libbuildpack/layers"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("go.mod") {
		ctx.OptOut("go.mod file not found")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("gopath")
	ctx.OverrideBuildEnv(l, "GOPATH", l.Root)
	ctx.OverrideBuildEnv(l, "GO111MODULE", "on")
	// Set GOPROXY to ensure no additional dependency is downloaded at built time.
	// All of them are downloaded here.
	ctx.OverrideBuildEnv(l, "GOPROXY", "off")
	ctx.WriteMetadata(l, nil, layers.Build)

	// TODO(b/145604612): Investigate caching the modules layer.

	// When there's a vendor folder and go is 1.14+, we shouldn't download the modules
	// and let go build use the vendored dependencies.
	if ctx.FileExists("vendor") {
		if golang.SupportsAutoVendor(ctx) {
			ctx.Logf("Not downloading modules because there's a `vendor` directory")
			return nil
		}

		ctx.Logf("Ignoring `vendor` directory: the Go runtime must be 1.14+ and go.mod should contain a `go 1.14`+ entry")
	}

	env := []string{"GOPATH=" + l.Root, "GO111MODULE=on"}
	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: []string{"go", "mod", "download"}, Env: env}, gcp.UserErrorKeepStderrTail)
	// go build -mod=readonly requires a complete graph of modules which `go mod download` does not produce in all cases (https://golang.org/issue/35832).
	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: []string{"go", "mod", "tidy"}, Env: env}, gcp.UserErrorKeepStderrTail)

	return nil
}
