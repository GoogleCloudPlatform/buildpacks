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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb"
)

const (
	layerName                     = "functions-framework"
	javaFunctionInvokerURLBase    = "https://maven-central.storage-download.googleapis.com/maven2/com/google/cloud/functions/invoker/java-function-invoker/"
	defaultFrameworkVersion       = "1.0.2"
	functionsFrameworkURLTemplate = javaFunctionInvokerURLBase + "%[1]s/java-function-invoker-%[1]s.jar"
	versionKey                    = "version"
	invokerMain                   = "com.google.cloud.functions.invoker.runner.Invoker"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	classpath, err := classpath(ctx)
	if err != nil {
		return err
	}

	layer := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	ffPath, err := installFunctionsFramework(ctx, layer)
	layer.BuildEnvironment.Override(java.FFJarPathEnv, ffPath)
	if err != nil {
		return err
	}

	ctx.SetFunctionsEnvVars(layer)

	// Use javap to check that the class is indeed in the classpath we just determined.
	// On success, it will output a description of the class and its public members, which we discard.
	// On failure it will output an error saying what's wrong (usually that the class doesn't exist).
	// Success here doesn't guarantee that the function will execute. It might not implement one of the
	// required interfaces, for example. But it eliminates the commonest problem of specifying the wrong target.
	// We use an ExecUser* method so that the time taken by the javap command is counted as user time.
	target := os.Getenv(env.FunctionTarget)
	if result, err := ctx.ExecWithErr([]string{"javap", "-classpath", classpath, target}, gcp.WithUserAttribution); err != nil {
		// The javap error output will typically be "Error: class not found: foo.Bar".
		return gcp.UserErrorf("build succeeded but did not produce the class %q specified as the function target: %s", target, result.Combined)
	}

	launcherSource := filepath.Join(ctx.BuildpackRoot(), "launch.sh")
	launcherTarget := filepath.Join(layer.Path, "launch.sh")
	createLauncher(ctx, launcherSource, launcherTarget)
	ctx.AddWebProcess([]string{launcherTarget, "java", "-jar", ffPath, "--classpath", classpath})

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
	if ctx.FileExists("build.gradle") {
		return gradleClasspath(ctx)
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

	mvn := java.MvnCmd(ctx)

	// Copy the dependencies of the function (`<dependencies>` in pom.xml) into target/dependency.
	ctx.Exec([]string{mvn, "--batch-mode", "dependency:copy-dependencies", "-Dmdep.prependGroupId", "-DincludeScope=runtime"}, gcp.WithUserAttribution)

	// Extract the final jar name from the user's pom.xml definitions.
	execResult := ctx.Exec([]string{mvn, "help:evaluate", "-q", "-DforceStdout", "-Dexpression=project.build.finalName"}, gcp.WithUserAttribution)
	artifactName := strings.TrimSpace(execResult.Stdout)
	if len(artifactName) == 0 {
		return "", gcp.UserErrorf("invalid project.build.finalName configured in pom.xml")
	}
	jarName := fmt.Sprintf("target/%s.jar", artifactName)
	if !ctx.FileExists(jarName) {
		return "", gcp.UserErrorf("expected output jar %s does not exist", jarName)
	}

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return jarName + ":target/dependency/*", nil
}

// gradleClasspath determines the --classpath when there is a build.gradle. This will consist of the jar file built
// from the build.gradle, plus all jar files that are dependencies mentioned there.
// Unlike Maven, Gradle doesn't have a simple way to query the contents of the build.gradle. But we can update
// the user's build.gradle to append tasks that do that. This is a bit ugly, but using --init-script didn't work
// because apparently you can't define tasks there; and having the predefined script include the user's build.gradle
// didn't work very well either, because you can't use a plugins {} clause in an included script.
func gradleClasspath(ctx *gcp.Context) (string, error) {
	extraTasksSource := filepath.Join(ctx.BuildpackRoot(), "extra_tasks.gradle")
	extraTasksText := ctx.ReadFile(extraTasksSource)
	if err := os.Chmod("build.gradle", 0644); err != nil {
		return "", gcp.InternalErrorf("making build.gradle writable: %v", err)
	}
	f, err := os.OpenFile("build.gradle", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return "", gcp.InternalErrorf("opening build.gradle for appending: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(extraTasksText); err != nil {
		return "", gcp.InternalErrorf("appending extra definitions to build.gradle: %v", err)
	}

	// Copy the dependencies of the function (`dependencies {...}` in build.gradle) into build/_javaFunctionDependencies.
	ctx.Exec([]string{"gradle", "--quiet", "_javaFunctionCopyAllDependencies"}, gcp.WithUserAttribution)

	// Extract the name of the target jar.
	execResult := ctx.Exec([]string{"gradle", "--quiet", "_javaFunctionPrintJarTarget"}, gcp.WithUserAttribution)
	jarName := strings.TrimSpace(execResult.Stdout)
	if !ctx.FileExists(jarName) {
		return "", gcp.UserErrorf("expected output jar %s does not exist", jarName)
	}

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return fmt.Sprintf("%s:build/_javaFunctionDependencies/*", jarName), nil
}

func installFunctionsFramework(ctx *gcp.Context, layer *libcnb.Layer) (string, error) {

	jars := []string{}
	if ctx.FileExists("pom.xml") {
		mvn := java.MvnCmd(ctx)
		// If the invoker was listed as a dependency in the pom.xml, copy it into target/_javaInvokerDependency.
		ctx.Exec([]string{
			mvn,
			"--batch-mode",
			"dependency:copy-dependencies",
			"-DoutputDirectory=target/_javaInvokerDependency",
			"-DincludeGroupIds=com.google.cloud.functions",
			"-DincludeArtifactIds=java-function-invoker",
		}, gcp.WithUserAttribution)
		jars = ctx.Glob("target/_javaInvokerDependency/java-function-invoker-*.jar")
	} else if ctx.FileExists("build.gradle") {
		// If the invoker was listed as an implementation dependency it will have been copied to build/_javaFunctionDependencies.
		jars = ctx.Glob("build/_javaFunctionDependencies/java-function-invoker-*.jar")
	}
	if len(jars) == 1 && isInvokerJar(ctx, jars[0]) {
		ctx.ClearLayer(layer)
		// No need to cache the layer because we aren't downloading the framework.
		layer.Cache = false
		return jars[0], nil
	}

	frameworkVersion := defaultFrameworkVersion

	// Install functions-framework.
	metaVersion := ctx.GetMetadata(layer, versionKey)
	if frameworkVersion == metaVersion {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)
		ctx.ClearLayer(layer)
		if err := installFramework(ctx, layer, frameworkVersion); err != nil {
			return "", err
		}
		ctx.SetMetadata(layer, versionKey, frameworkVersion)
	}
	return filepath.Join(layer.Path, "functions-framework.jar"), nil
}

// isInvokerjar checks if the .jar at the given filepath is the functions framework invoker by checking
// that the manifest's Main-Class matches an expected value.
func isInvokerJar(ctx *gcp.Context, jar string) bool {
	main, err := java.MainManifestEntry(jar)
	ctx.Warnf("Failed to identify functions framework invoker dependency. Installing version %s:\n%v", defaultFrameworkVersion, err)
	return err == nil && main == invokerMain
}

// installFramework downloads the functions framework invoker jar and saves it in the provided layer.
func installFramework(ctx *gcp.Context, layer *libcnb.Layer, version string) error {
	url := fmt.Sprintf(functionsFrameworkURLTemplate, version)
	ffName := filepath.Join(layer.Path, "functions-framework.jar")
	result, err := ctx.ExecWithErr([]string{"curl", "--silent", "--fail", "--show-error", "--output", ffName, url})
	// We use ExecWithErr rather than plain Exec because if it fails we want to exit with an error message better
	// than "Failure: curl: (22) The requested URL returned error: 404".
	// TODO(b/155874677): use plain Exec once it gives sufficient error messages.
	if err != nil {
		return gcp.InternalErrorf("fetching functions framework jar: %v\n%s", err, result.Stderr)
	}
	return nil
}
