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
	"os"
	"path/filepath"
	"strconv"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	archiveName = "source-code.tar.gz"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	// Fail archiving source when users want to clear source from the final container.
	if cs, ok := os.LookupEnv(env.ClearSource); ok {
		c, err := strconv.ParseBool(cs)
		if err != nil {
			ctx.Warnf("Failed to parse %q: %v", env.ClearSource, err)
		} else if c {
			ctx.OptOut("%s is incompatible with archive source", env.ClearSource)
		}
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	sl := ctx.Layer("src", gcp.LaunchLayer)
	sp := filepath.Join(sl.Path, archiveName)
	archiveSource(ctx, sp, ctx.ApplicationRoot())

	// Symlink the archive to /workspace/.googlebuild for a stable path; add LABEL to container.
	ctx.MkdirAll(".googlebuild", 0755)
	stable := filepath.Join(ctx.ApplicationRoot(), ".googlebuild", archiveName)
	ctx.Symlink(sp, stable)
	ctx.AddLabel("source-archive", stable)

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
