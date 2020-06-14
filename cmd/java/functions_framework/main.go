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

// Implements java/functions_framework buildpack.
// The functions_framework buildpack copies the function framework into a layer, and adds it to a compiled function to make an executable app.
package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	layer := ctx.Layer(layerName)

	if err := installFunctionsFramework(ctx, layer); err != nil {
		return err
	}

	classpath, err := classpath(ctx)
	if err != nil {
		return err
	}

	ctx.SetFunctionsEnvVars(layer)

	launcherSource := filepath.Join(ctx.BuildpackRoot(), "launch.sh")
	launcherTarget := filepath.Join(layer.Root, "launch.sh")
	createLauncher(ctx, launcherSource, launcherTarget)
	ctx.AddWebProcess([]string{launcherTarget, "java", "-jar", filepath.Join(layer.Root, "functions-framework.jar"), "--classpath", classpath})

	return nil
}

func createLauncher(ctx *gcp.Context, launcherSource, launcherTarget string) {
	launcherContents := ctx.ReadFile(launcherSource)
	ctx.WriteFile(launcherTarget, launcherContents, 0755)
}

// classpath determines what the --classpath argument should be. This tells the Functions Framework where to find
// the classes of the function, including dependencies.
func classpath(ctx *gcp.Context) (string, error) {
	if ctx.FileExists("pom.xml") {
		return mavenClasspath(ctx)
	}
	jars := ctx.Glob("*.jar")
	if len(jars) == 1 {
		// Already-built jar file. It should be self-contained, which means that it can be the only thing given to --classpath.
		return jars[0], nil
	}
	if len(jars) > 1 {
		return "", gcp.UserErrorf("function has no pom.xml and more than one jar file: %s", strings.Join(jars, ", "))
	}
	// We have neither pom.xml nor a jar file. Show what files there are. If the user deployed the wrong directory, this may help them see the problem more easily.
	description := "directory is empty"
	if files := ctx.Glob("*"); len(files) > 0 {
		description = fmt.Sprintf("directory has these entries: %s", strings.Join(files, ", "))
	}
	return "", gcp.UserErrorf("function has neither pom.xml nor already-built jar file; %s", description)
}

// mavenClasspath determines the --classpath when there is a pom.xml. This will consist of the jar file built
// from the pom.xml itself, plus all jar files that are dependencies mentioned in the pom.xml.
func mavenClasspath(ctx *gcp.Context) (string, error) {
	// Copy the dependencies of the function (<dependencies> in pom.xml) into target/dependency.
	ctx.ExecUser([]string{"mvn", "dependency:copy-dependencies"})

	// Extract the artifact/version coordinates from the user's pom.xml definitions.
	// mvn help:evaluate is quite slow so we do it this way rather than calling it twice.
	// The name of the built jar file will be <artifact>-<version>.jar, for example myfunction-0.9.jar.
	execResult := ctx.ExecUser([]string{"mvn", "help:evaluate", "-q", "-DforceStdout", "-Dexpression=project.artifactId/${project.version}"})
	groupArtifactVersion := execResult.Stdout
	components := strings.Split(groupArtifactVersion, "/")
	if len(components) != 2 {
		return "", gcp.UserErrorf("could not parse query output into artifact/version: %s", groupArtifactVersion)
	}
	artifact, version := components[0], components[1]

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return fmt.Sprintf("target/%s-%s.jar:target/dependency/*", artifact, version), nil
}

func installFunctionsFramework(ctx *gcp.Context, layer *layers.Layer) error {
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
