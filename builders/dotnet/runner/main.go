// The runner binary executes buildpacks for the .NET language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	dotnetappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/appengine/lib"
	dotnetappenginemain "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/appengine_main/lib"
	dotnetflex "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/flex/lib"
	dotnetfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/functions_framework/lib"
	dotnetpublish "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/publish/lib"
	dotnetruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/runtime/lib"
	dotnetsdk "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/sdk/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.dotnet.appengine"] = gcp.BuildpackFuncs{
		Detect: dotnetappengine.DetectFn,
		Build:  dotnetappengine.BuildFn,
	}
	buildpacks["google.dotnet.appengine-main"] = gcp.BuildpackFuncs{
		Detect: dotnetappenginemain.DetectFn,
		Build:  dotnetappenginemain.BuildFn,
	}
	buildpacks["google.dotnet.flex"] = gcp.BuildpackFuncs{
		Detect: dotnetflex.DetectFn,
		Build:  dotnetflex.BuildFn,
	}
	buildpacks["google.dotnet.runtime"] = gcp.BuildpackFuncs{
		Detect: dotnetruntime.DetectFn,
		Build:  dotnetruntime.BuildFn,
	}
	buildpacks["google.dotnet.sdk"] = gcp.BuildpackFuncs{
		Detect: dotnetsdk.DetectFn,
		Build:  dotnetsdk.BuildFn,
	}
	buildpacks["google.dotnet.publish"] = gcp.BuildpackFuncs{
		Detect: dotnetpublish.DetectFn,
		Build:  dotnetpublish.BuildFn,
	}
	buildpacks["google.dotnet.functions-framework"] = gcp.BuildpackFuncs{
		Detect: dotnetfunctionsframework.DetectFn,
		Build:  dotnetfunctionsframework.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/dotnet/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
