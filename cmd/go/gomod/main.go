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
	"fmt"
	"os"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

func main() {
	gcp.Main(DetectFn, BuildFn)
}

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	goModExists, err := ctx.FileExists("go.mod")
	if err != nil {
		return nil, err
	}
	if goModExists {
		return gcp.OptInFileFound("go.mod"), nil
	}
	return gcp.OptOutFileNotFound("go.mod"), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	l, err := golang.NewGoWorkspaceLayer(ctx)
	if err != nil {
		return fmt.Errorf("creating GOPATH layer: %w", err)
	}

	vendorExists, err := ctx.FileExists("vendor")
	if err != nil {
		return err
	}
	// When there's a vendor folder and go is 1.14+, we shouldn't download the modules
	// and let go build use the vendored dependencies.
	if vendorExists {
		avSupport, err := golang.SupportsAutoVendor(ctx)
		if err != nil {
			return fmt.Errorf("checking for auto vendor support: %w", err)
		}
		if avSupport {
			ctx.Logf("Not downloading modules because there's a `vendor` directory")
			return nil
		}

		ctx.Warnf(`Ignoring "vendor" directory: To use vendor directory, the Go runtime must be 1.14+ and go.mod must contain a "go 1.14"+ entry. See https://cloud.google.com/appengine/docs/standard/go/specifying-dependencies#vendoring_dependencies.`)
	}

	goModIsWriteable, err := ctx.IsWritable("go.mod")
	if err != nil {
		return err
	}
	if !goModIsWriteable {
		// Preempt an obscure failure mode: if go.mod is not writable then `go list -m` can fail saying:
		//     go: updates to go.sum needed, disabled by -mod=readonly
		return gcp.UserErrorf("go.mod exists but is not writable")
	}
	env := []string{"GOPATH=" + l.Path, "GO111MODULE=on"}

	// BuildDirEnv should only be set by App Engine buildpacks.
	workdir := os.Getenv(golang.BuildDirEnv)
	if workdir == "" {
		workdir = ctx.ApplicationRoot()
	}

	goSumExists, err := ctx.FileExists("go.sum")
	if err != nil {
		return err
	}
	// Go 1.16+ requires a go.sum file. If one does not exist, generate it.
	// go build -mod=readonly requires a complete graph of modules which `go mod download` does not produce in all cases (https://golang.org/issue/35832).
	if !goSumExists {
		ctx.Logf(`go.sum not found, generating using "go mod tidy"`)
		if _, err := golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "tidy"}, gcp.WithEnv(env...), gcp.WithWorkDir(workdir), gcp.WithUserAttribution); err != nil {
			return fmt.Errorf("running go mod tidy: %w", err)
		}
	}

	if _, err := golang.ExecWithGoproxyFallback(ctx, []string{"go", "mod", "download"}, gcp.WithEnv(env...), gcp.WithUserAttribution); err != nil {
		return fmt.Errorf("running go mod download: %w", err)
	}

	return nil
}
