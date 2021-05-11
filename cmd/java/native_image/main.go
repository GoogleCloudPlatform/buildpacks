// Copyright 2021 Google LLC
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

// Implements Java GraalVM Native Image buildpack.
// This buildpack installs the GraalVM compiler into a layer and builds a native image of the Java application.
package main

import (
	"fmt"
	"path"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb"
)

var (
	requiresGraalvm = []libcnb.BuildPlanRequire{{Name: "graalvm"}}
	planRequires    = libcnb.BuildPlan{Requires: requiresGraalvm}
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(gcp.WithBuildPlans(planRequires)), nil
}

func buildFn(ctx *gcp.Context) error {
	jar, err := java.ExecutableJar(ctx)
	if err != nil {
		return fmt.Errorf("finding executable jar: %w", err)
	}

	nativeLayer := ctx.Layer("native-image", gcp.LaunchLayer)
	// Not using ctx.TempDir(), as moving files from /tmp to a different volume will fail.
	tempLayer := ctx.Layer("temp")
	imagePath := filepath.Join(tempLayer.Path, "native-app")

	// This may generate extra files (*.o and *.build_artifacts.txt) alongside.
	command := []string{
		"native-image",
		"--no-fallback", "--no-server",
		"-H:+StaticExecutableWithDynamicLibC",
		"-jar", jar,
		imagePath}
	if _, err := ctx.ExecWithErr(command, gcp.WithUserAttribution); err != nil {
		return err
	}

	finalImage := filepath.Join(nativeLayer.Path, "bin", "native-app")
	ctx.MkdirAll(path.Dir(finalImage), 0744)
	ctx.Rename(imagePath, finalImage)

	ctx.AddWebProcess([]string{"native-app"})
	return nil
}
