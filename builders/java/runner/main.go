// The runner binary executes buildpacks for the Java language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	javaappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/java/appengine/lib"
	javaclearsource "github.com/GoogleCloudPlatform/buildpacks/cmd/java/clear_source/lib"
	javaentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/java/entrypoint/lib"
	javaexplodedjar "github.com/GoogleCloudPlatform/buildpacks/cmd/java/exploded_jar/lib"
	javafunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/java/functions_framework/lib"
	javagradle "github.com/GoogleCloudPlatform/buildpacks/cmd/java/gradle/lib"
	javamaven "github.com/GoogleCloudPlatform/buildpacks/cmd/java/maven/lib"
	javaruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/java/runtime/lib"
	javaspringboot "github.com/GoogleCloudPlatform/buildpacks/cmd/java/spring_boot/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.java.appengine"] = gcp.BuildpackFuncs{
		Detect: javaappengine.DetectFn,
		Build:  javaappengine.BuildFn,
	}
	buildpacks["google.java.clear-source"] = gcp.BuildpackFuncs{
		Detect: javaclearsource.DetectFn,
		Build:  javaclearsource.BuildFn,
	}
	buildpacks["google.java.entrypoint"] = gcp.BuildpackFuncs{
		Detect: javaentrypoint.DetectFn,
		Build:  javaentrypoint.BuildFn,
	}
	buildpacks["google.java.exploded-jar"] = gcp.BuildpackFuncs{
		Detect: javaexplodedjar.DetectFn,
		Build:  javaexplodedjar.BuildFn,
	}
	buildpacks["google.java.functions-framework"] = gcp.BuildpackFuncs{
		Detect: javafunctionsframework.DetectFn,
		Build:  javafunctionsframework.BuildFn,
	}
	buildpacks["google.java.gradle"] = gcp.BuildpackFuncs{
		Detect: javagradle.DetectFn,
		Build:  javagradle.BuildFn,
	}
	buildpacks["google.java.maven"] = gcp.BuildpackFuncs{
		Detect: javamaven.DetectFn,
		Build:  javamaven.BuildFn,
	}
	buildpacks["google.java.runtime"] = gcp.BuildpackFuncs{
		Detect: javaruntime.DetectFn,
		Build:  javaruntime.BuildFn,
	}
	buildpacks["google.java.spring-boot"] = gcp.BuildpackFuncs{
		Detect: javaspringboot.DetectFn,
		Build:  javaspringboot.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/java/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
