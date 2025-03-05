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

// Implements java/functions_framework buildpack.
// The functions_framework buildpack copies the function framework into a layer, and adds it to a compiled function to make an executable app.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cloudfunctions"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/java"
	"github.com/buildpacks/libcnb/v2"
)

const (
	layerName                     = "functions-framework"
	javaFunctionInvokerURLBase    = "https://maven-central.storage-download.googleapis.com/maven2/com/google/cloud/functions/invoker/java-function-invoker/"
	defaultFrameworkVersion       = "1.3.3"
	functionsFrameworkURLTemplate = javaFunctionInvokerURLBase + "%[1]s/java-function-invoker-%[1]s.jar"
	versionKey                    = "version"
	invokerMain                   = "com.google.cloud.functions.invoker.runner.Invoker"
	implementationVersionKey      = "Implementation-Version"
)

var (
	frameworkVersionRegex = regexp.MustCompile("java-function-invoker-((\\d+\\.)*\\d+)")
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

	layer, err := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}
	ffPath, err := installFunctionsFramework(ctx, layer)
	layer.BuildEnvironment.Override(java.FFJarPathEnv, ffPath)
	if err != nil {
		return err
	}

	if err := ctx.SetFunctionsEnvVars(layer); err != nil {
		return err
	}

	// Use javap to check that the class is indeed in the classpath we just determined.
	// On success, it will output a description of the class and its public members, which we discard.
	// On failure it will output an error saying what's wrong (usually that the class doesn't exist).
	// Success here doesn't guarantee that the function will execute. It might not implement one of the
	// required interfaces, for example. But it eliminates the commonest problem of specifying the wrong target.
	// We use an ExecUser* method so that the time taken by the javap command is counted as user time.
	target := os.Getenv(env.FunctionTarget)
	if result, err := ctx.Exec([]string{"javap", "-classpath", classpath, target}, gcp.WithUserAttribution); err != nil {
		// The javap error output will typically be "Error: class not found: foo.Bar".
		return gcp.UserErrorf("build succeeded but did not produce the class %q specified as the function target: %s", target, result.Combined)
	}

	launcherSource := filepath.Join(ctx.BuildpackRoot(), "launch.sh")
	launcherTarget := filepath.Join(layer.Path, "launch.sh")
	createLauncher(ctx, launcherSource, launcherTarget)
	ctx.AddWebProcess([]string{launcherTarget, "java", "-jar", ffPath, "--classpath", classpath})

	return nil
}

func createLauncher(ctx *gcp.Context, launcherSource, launcherTarget string) error {
	launcherContents, err := ctx.ReadFile(launcherSource)
	if err != nil {
		return err
	}
	if err := ctx.WriteFile(launcherTarget, launcherContents, os.FileMode(0755)); err != nil {
		return err
	}
	return nil
}

// classpath determines what the --classpath argument should be. This tells the Functions Framework where to find
// the classes of the function, including dependencies.
func classpath(ctx *gcp.Context) (string, error) {
	pomExists, err := ctx.FileExists("pom.xml")
	if err != nil {
		return "", err
	}
	if pomExists {
		return mavenClasspath(ctx)
	}
	buildGradleExists, err := ctx.FileExists("build.gradle")
	if err != nil {
		return "", err
	}
	if buildGradleExists {
		return gradleClasspath(ctx)
	}
	jars, err := ctx.Glob("*.jar")
	if err != nil {
		return "", fmt.Errorf("finding jar files: %w", err)
	}
	if len(jars) == 1 {
		// Already-built jar file. It should be self-contained, which means that it can be the only thing given to --classpath.
		return jars[0], nil
	}
	if len(jars) > 1 {
		return "", gcp.UserErrorf("function has no pom.xml and more than one jar file: %s", strings.Join(jars, ", "))
	}
	// We have neither pom.xml nor a jar file. Show what files there are. If the user deployed the wrong directory, this may help them see the problem more easily.
	description := "directory is empty"
	files, err := ctx.Glob("*")
	if err != nil {
		return "", fmt.Errorf("finding files: %w", err)
	}
	if len(files) > 0 {
		description = fmt.Sprintf("directory has these entries: %s", strings.Join(files, ", "))
	}
	return "", gcp.UserErrorf("function has neither pom.xml nor already-built jar file; %s", description)
}

// mavenClasspath determines the --classpath when there is a pom.xml. This will consist of the jar file built
// from the pom.xml itself, plus all jar files that are dependencies mentioned in the pom.xml.
func mavenClasspath(ctx *gcp.Context) (string, error) {
	mvn, err := java.MvnCmd(ctx)
	if err != nil {
		return "", err
	}

	// Copy the dependencies of the function (`<dependencies>` in pom.xml) into target/dependency.
	if _, err := ctx.Exec([]string{mvn, "--batch-mode", "dependency:copy-dependencies", "-Dmdep.prependGroupId", "-DincludeScope=runtime"}, gcp.WithUserAttribution); err != nil {
		return "", err
	}

	// Extract the final jar name from the user's pom.xml definitions.
	execResult, err := ctx.Exec([]string{mvn, "help:evaluate", "-q", "-DforceStdout", "-Dexpression=project.build.finalName"}, gcp.WithUserAttribution)
	if err != nil {
		return "", err
	}
	artifactName := strings.TrimSpace(execResult.Stdout)
	if len(artifactName) == 0 {
		return "", gcp.UserErrorf("invalid project.build.finalName configured in pom.xml")
	}
	jarName := fmt.Sprintf("target/%s.jar", artifactName)
	jarExists, err := ctx.FileExists(jarName)
	if err != nil {
		return "", err
	}
	if !jarExists {
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
	gradle, err := java.GradleCmd(ctx)
	if err != nil {
		return "", err
	}

	extraTasksSource := filepath.Join(ctx.BuildpackRoot(), "extra_tasks.gradle")
	extraTasksText, err := ctx.ReadFile(extraTasksSource)
	if err != nil {
		return "", err
	}
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
	if _, err := ctx.Exec([]string{gradle, "--quiet", "_javaFunctionCopyAllDependencies"}, gcp.WithUserAttribution); err != nil {
		return "", err
	}

	// Extract the name of the target jar.
	execResult, err := ctx.Exec([]string{gradle, "--quiet", "_javaFunctionPrintJarTarget"}, gcp.WithUserAttribution)
	if err != nil {
		return "", err
	}
	jarName := strings.TrimSpace(execResult.Stdout)
	jarExists, err := ctx.FileExists(jarName)
	if err != nil {
		return "", err
	}
	if !jarExists {
		return "", gcp.UserErrorf("expected output jar %s does not exist", jarName)
	}

	// The Functions Framework understands "*" to mean every jar file in that directory.
	// So this classpath consists of the just-built jar and all of the dependency jars.
	return fmt.Sprintf("%s:build/_javaFunctionDependencies/*", jarName), nil
}

func installFunctionsFramework(ctx *gcp.Context, layer *libcnb.Layer) (string, error) {

	jars := []string{}
	pomExists, err := ctx.FileExists("pom.xml")
	if err != nil {
		return "", err
	}
	if pomExists {
		mvn, err := java.MvnCmd(ctx)
		if err != nil {
			return "", err
		}
		// If the invoker was listed as a dependency in the pom.xml, copy it into target/_javaInvokerDependency.
		if _, err := ctx.Exec([]string{
			mvn,
			"--batch-mode",
			"dependency:copy-dependencies",
			"-DoutputDirectory=target/_javaInvokerDependency",
			"-DincludeGroupIds=com.google.cloud.functions",
			"-DincludeArtifactIds=java-function-invoker",
		}, gcp.WithUserAttribution); err != nil {
			return "", err
		}
		jars, err = ctx.Glob("target/_javaInvokerDependency/java-function-invoker-*.jar")
		if err != nil {
			return "", fmt.Errorf("finding java-function-invoker jar: %w", err)
		}
	} else {
		buildGradleExists, err := ctx.FileExists("build.gradle")
		if err != nil {
			return "", err
		}
		if buildGradleExists {
			// If the invoker was listed as an implementation dependency it will have been copied to build/_javaFunctionDependencies.
			jars, err = ctx.Glob("build/_javaFunctionDependencies/java-function-invoker-*.jar")
			if err != nil {
				return "", fmt.Errorf("finding java-function-invoker jar: %w", err)
			}
		}
	}
	if len(jars) == 1 && isInvokerJar(ctx, jars[0]) {
		if err := ctx.ClearLayer(layer); err != nil {
			return "", fmt.Errorf("clearing layer %q: %w", layer.Name, err)
		}
		// No need to cache the layer because we aren't downloading the framework.
		layer.Cache = false
		addFrameworkVersionLabel(ctx, layer, jars[0])
		return jars[0], nil
	}
	ctx.Warnf("Failed to find vendored functions-framework dependency. Installing version %s:\n%v", defaultFrameworkVersion, err)

	frameworkVersion := defaultFrameworkVersion

	// Install functions-framework.
	metaVersion := ctx.GetMetadata(layer, versionKey)
	if frameworkVersion == metaVersion {
		ctx.CacheHit(layerName)
	} else {
		ctx.CacheMiss(layerName)
		if err := ctx.ClearLayer(layer); err != nil {
			return "", fmt.Errorf("clearing layer %q: %w", layer.Name, err)
		}
		if err := downloadFramework(ctx, layer, frameworkVersion); err != nil {
			return "", err
		}
		ctx.SetMetadata(layer, versionKey, frameworkVersion)
	}
	cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
		Runtime:  "java",
		Version:  frameworkVersion,
		Injected: true,
	})

	return filepath.Join(layer.Path, "functions-framework.jar"), nil
}

// isInvokerjar checks if the .jar at the given filepath is the functions framework invoker by checking
// that the manifest's Main-Class matches an expected value.
func isInvokerJar(ctx *gcp.Context, jar string) bool {
	main, err := java.MainManifestEntry(jar)
	return err == nil && main == invokerMain
}

func addFrameworkVersionLabel(ctx *gcp.Context, layer *libcnb.Layer, frameworkJar string) {
	version, err := java.FindManifestValueFromJar(frameworkJar, implementationVersionKey)
	if err != nil {
		ctx.Logf("Functions framework manifest could not be read: %v", err)
	}
	if version == "" {
		// If version isn't found the ff version may predate setting implementationVersionKey.
		// In these cases a regex match is the best way to identify the framework version.
		if matches := frameworkVersionRegex.FindStringSubmatch(frameworkJar); matches != nil {
			version = matches[1]
		} else {
			ctx.Logf("Unable to identify functions framework version from %v", frameworkJar)
			version = "unknown"
		}
	}
	cloudfunctions.AddFrameworkVersionLabel(ctx, &cloudfunctions.FrameworkVersionInfo{
		Runtime:  "java",
		Version:  version,
		Injected: false,
	})
}

// downloadFramework downloads the functions framework invoker jar and saves it in the provided layer.
func downloadFramework(ctx *gcp.Context, layer *libcnb.Layer, version string) error {
	url := fmt.Sprintf(functionsFrameworkURLTemplate, version)
	ffName := filepath.Join(layer.Path, "functions-framework.jar")
	result, err := ctx.Exec([]string{"curl", "--silent", "--fail", "--show-error", "--output", ffName, url})
	if err != nil {
		return gcp.InternalErrorf("fetching functions framework jar: %v\n%s", err, result.Stderr)
	}
	return nil
}
