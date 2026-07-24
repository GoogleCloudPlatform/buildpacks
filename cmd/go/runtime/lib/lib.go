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

// Implements go/runtime buildpack.
// The runtime buildpack installs the Go toolchain.
package lib

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	goLayer      = "go"
	envGoVersion = "GOOGLE_GO_VERSION"
)

// DetectFn detects if .go files are present.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("go"); result != nil {
		return result, nil
	}
	atLeastOne, err := ctx.HasAtLeastOneOutsideDependencyDirectories("*.go")
	if err != nil {
		return nil, fmt.Errorf("finding *.go files: %w", err)
	}
	if atLeastOne {
		return gcp.OptIn("found .go files"), nil
	}
	return gcp.OptOut("no .go files found"), nil
}

// BuildFn installs the Go toolchain.
func BuildFn(ctx *gcp.Context) error {
	version, err := golang.RuntimeVersion(ctx)
	if err != nil {
		return err
	}
	grl, err := ctx.Layer(goLayer, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}
	_, err = runtime.InstallTarballIfNotCached(ctx, runtime.Go, version, grl)
	return err
}
