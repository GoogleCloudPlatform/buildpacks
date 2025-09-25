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

// Implements the java/entrypoint buildpack.
package lib

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	manifestExists, err := ctx.FileExists(java.ManifestPath)
	if err != nil {
		return nil, err
	}
	if manifestExists {
		return gcp.OptInFileFound(java.ManifestPath), nil
	}
	return gcp.OptOutFileNotFound(java.ManifestPath), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	main, err := java.MainFromManifest(ctx, java.ManifestPath)
	if err != nil {
		return fmt.Errorf("extracting Main-Class from %s: %w", java.ManifestPath, err)
	}
	ctx.AddWebProcess([]string{"java", "-classpath", ".", main})
	return nil
}
