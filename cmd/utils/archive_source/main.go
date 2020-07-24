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

// Implements utils/archive-source buildpack.
// The archive-source buildpack archives user's source code.
package main

import (
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	archiveName = "source-code.tar.gz"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	return nil
}

func buildFn(ctx *gcp.Context) error {
	sl := ctx.Layer("src")
	sp := filepath.Join(sl.Root, archiveName)
	archiveSource(ctx, sp, ctx.ApplicationRoot())

	// Symlink the archive to /workspace/.googlebuild for a stable path.
	ctx.MkdirAll(".googlebuild", 0755)
	ctx.Symlink(sp, filepath.Join(ctx.ApplicationRoot(), ".googlebuild", archiveName))

	ctx.WriteMetadata(sl, nil, layers.Launch)

	return nil
}

// archiveSource archives user's source code in a layer
func archiveSource(ctx *gcp.Context, fileName, dirName string) {
	ctx.Exec([]string{"tar",
		"--create", "--gzip", "--preserve-permissions",
		"--file=" + fileName,
		"--directory", dirName,
		"."}, gcp.WithUserTimingAttribution)
}
