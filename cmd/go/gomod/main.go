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
	"os"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if ctx.FileExists("go.mod") {
		return gcp.OptInFileFound("go.mod"), nil
	}
	return gcp.OptOutFileNotFound("go.mod"), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer("gopath", gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	l.BuildEnvironment.Override("GOPATH", l.Path)
	l.BuildEnvironment.Override("GO111MODULE", "on")
	// Set GOPROXY to ensure no additional dependency is downloaded at built time.
	// All of them are downloaded here.
	l.BuildEnvironment.Override("GOPROXY", "off")

	// TODO(b/145604612): Investigate caching the modules layer.

	// When there's a vendor folder and go is 1.14+, we shouldn't download the modules
	// and let go build use the vendored dependencies.
	if ctx.FileExists("vendor") {
		if golang.SupportsAutoVendor(ctx) {
			ctx.Logf("Not downloading modules because there's a `vendor` directory")
			return nil
		}

		ctx.Warnf(`Ignoring "vendor" directory: To use vendor directory, the Go runtime must be 1.14+ and go.mod must contain a "go 1.14"+ entry. See https://cloud.google.com/appengine/docs/standard/go/specifying-dependencies#vendoring_dependencies.`)
	}

	if info, err := os.Stat("go.mod"); err == nil && info.Mode().Perm()&0200 == 0 {
		return gcp.UserErrorf("go.mod exists but is not writable")
	}
	env := []string{"GOPATH=" + l.Path, "GO111MODULE=on"}

	// BuildDirEnv should only be set by App Engine buildpacks.
	workdir := os.Getenv(golang.BuildDirEnv)
	if workdir == "" {
		workdir = ctx.ApplicationRoot()
	}

	// Go 1.16+ requires a go.sum file. If one does not exist, generate it.
	// go build -mod=readonly requires a complete graph of modules which `go mod download` does not produce in all cases (https://golang.org/issue/35832).
	if !ctx.FileExists("go.sum") {
		golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "tidy"}, gcp.WithEnv(env...), gcp.WithWorkDir(workdir), gcp.WithUserAttribution)
	}

	golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "download"}, gcp.WithEnv(env...), gcp.WithUserAttribution)

	return nil
}
