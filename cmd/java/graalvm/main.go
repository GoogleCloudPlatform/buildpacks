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

// Implements Java GraalVM Native Image buildpack.
// This buildpack installs the GraalVM compiler into a layer and builds a native image of the Java application.
package main

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

const (
	// TODO(mpeddada): Upgrade the GraalVM version. The version has currently been
	// downgraded from 21.1.0 as building a native image for a standard GCF
	// workflow by calling `native-image -cp ...` was resulting in a parsing error.
	graalvmVersion = "21.0.0"
	graalvmURL     = "https://github.com/graalvm/graalvm-ce-builds/releases/download/vm-%[1]s/graalvm-ce-java11-linux-amd64-%[1]s.tar.gz"
	layerName      = "java-graalvm"
	versionKey     = "version"
)

var (
	providesGraalvm = []libcnb.BuildPlanProvide{{Name: "graalvm"}}
	planProvides    = libcnb.BuildPlan{Provides: providesGraalvm}
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	useNativeImage, err := env.IsUsingNativeImage()
	if err != nil {
		return nil, gcp.UserErrorf("failed to parse GOOGLE_JAVA_USE_NATIVE_IMAGE: %v", err)
	}

	if useNativeImage {
		ctx.Warnf("The GraalVM Native Image buildpack is enabled. Note: This is under development and not ready for use.")
		return gcp.OptInEnvSet(env.UseNativeImage, gcp.WithBuildPlans(planProvides)), nil
	}

	return gcp.OptOutEnvNotSet(env.UseNativeImage), nil
}

func buildFn(ctx *gcp.Context) error {
	if err := installGraalVM(ctx); err != nil {
		return err
	}

	return nil
}

func installGraalVM(ctx *gcp.Context) error {
	graalLayer, err := ctx.Layer(layerName, gcp.CacheLayer, gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", graalLayer, err)
	}

	metaVersion := ctx.GetMetadata(graalLayer, versionKey)
	if graalvmVersion == metaVersion {
		ctx.CacheHit(layerName)
		ctx.Logf("GraalVM cache hit, skipping installation.")
		return nil
	}

	ctx.CacheMiss(layerName)
	if err := ctx.ClearLayer(graalLayer); err != nil {
		return fmt.Errorf("clearing layer %q: %w", graalLayer.Name, err)
	}

	// Install graalvm into layer.
	archiveURL := fmt.Sprintf(graalvmURL, graalvmVersion)
	command := fmt.Sprintf(
		"curl --fail --show-error --silent --location %s "+
			"| tar xz --directory %s --strip-components=1", archiveURL, graalLayer.Path)
	if _, err := ctx.Exec([]string{"bash", "-c", command}, gcp.WithUserAttribution); err != nil {
		return err
	}

	// Install native-image component
	graalUpdater := filepath.Join(graalLayer.Path, "bin", "gu")
	_, err = ctx.Exec([]string{graalUpdater, "install", "native-image"}, gcp.WithUserAttribution)
	if err != nil {
		return err
	}

	ctx.SetMetadata(graalLayer, versionKey, graalvmVersion)
	return nil
}
