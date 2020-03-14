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

// Implements /bin/build for go/gomod buildpack.
package main

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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
	ctx.WriteMetadata(l, nil, layers.Build)

	// TODO: cache the modules layer once you prove that the latency of re-dding a large layer is less than the latency of downloading from the internet
	// TODO: cache the GOCACHE layer once you prove that the latency of re-dding a large layer is less than the latency of building without GOCACHE

	env := []string{"GOPATH=" + l.Root, "GO111MODULE=on"}
	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: []string{"go", "mod", "download"}, Env: env}, gcp.UserErrorKeepStderrTail)
	// go build -mod=readonly requires a complete graph of modules which `go mod download` does not produce in all cases (https://golang.org/issue/35832).
	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: []string{"go", "mod", "tidy"}, Env: env}, gcp.UserErrorKeepStderrTail)

	return nil
}
