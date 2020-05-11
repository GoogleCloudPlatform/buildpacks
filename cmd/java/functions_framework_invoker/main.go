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

// Implements java/functions_framework_invoker buildpack.
// The functions_framework_invoker buildpack copies the function framework into a layer.
package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	layerName                     = "functions-framework"
	javaFunctionInvokerURLBase    = "https://maven-central.storage-download.googleapis.com/maven2/com/google/cloud/functions/invoker/java-function-invoker/"
	defaultFrameworkVersion       = "1.0.0-alpha-2-rc5"
	functionsFrameworkMetadataURL = javaFunctionInvokerURLBase + "maven-metadata.xml"
	functionsFrameworkURLTemplate = javaFunctionInvokerURLBase + "%[1]s/java-function-invoker-%[1]s.jar"
)

// metadata represents metadata stored for the functions framework layer.
type metadata struct {
	Version string `toml:"version"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		ctx.OptIn("%s set", env.FunctionTarget)
	}
	ctx.OptOut("%s not set", env.FunctionTarget)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	frameworkVersion := defaultFrameworkVersion
	// TODO(emcmanus): extract framework version from pom.xml if present
	if version, err := latestFrameworkVersion(ctx); err == nil {
		frameworkVersion = version
		ctx.Logf("Using latest framework version %s", version)
	} else {
		ctx.Warnf("Could not determine latest framework version, defaulting to %s: %v", defaultFrameworkVersion, err)
	}

	// Install functions-framework.
	var meta metadata
	layer := ctx.Layer(layerName)
	ctx.ReadMetadata(layer, &meta)
	if frameworkVersion == meta.Version {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)
		ctx.ClearLayer(layer)
		if err := installFramework(ctx, layer, frameworkVersion); err != nil {
			return err
		}
		meta.Version = frameworkVersion

		ctx.WriteMetadata(layer, meta, layers.Launch, layers.Cache)
	}

	ctx.SetFunctionsEnvVars(layer)

	return nil
}

func installFramework(ctx *gcp.Context, layer *layers.Layer, version string) error {
	url := fmt.Sprintf(functionsFrameworkURLTemplate, version)
	ffName := filepath.Join(layer.Root, "functions-framework.jar")
	result, err := ctx.ExecWithErr([]string{"curl", "--silent", "--fail", "--show-error", "--output", ffName, url})
	// We use ExecWithErr rather than plain Exec because if it fails we want to exit with an error message better
	// than "Failure: curl: (22) The requested URL returned error: 404".
	// TODO(b/155874677): use plain Exec once it gives sufficient error messages.
	if err != nil {
		return gcp.InternalErrorf("fetching functions framework jar: %v\n%s", err, result.Stderr)
	}
	return nil
}

type mavenMetadata struct {
	XMLName xml.Name `xml:"metadata"`
	Release string   `xml:"versioning>release"`
}

func latestFrameworkVersion(ctx *gcp.Context) (string, error) {
	result, err := ctx.ExecWithErr([]string{"curl", "--silent", "--fail", "--show-error", functionsFrameworkMetadataURL})
	if err != nil {
		return "", gcp.InternalErrorf("fetching latest version: %v\n%s", err, result.Stderr)
	}
	metadataXML := result.Stdout
	var mavenMetadata mavenMetadata
	if err := xml.Unmarshal([]byte(metadataXML), &mavenMetadata); err != nil {
		return "", gcp.InternalErrorf("decoding release version in text from %s: %v:\n%s", functionsFrameworkMetadataURL, err, metadataXML)
	}
	return mavenMetadata.Release, nil
}
