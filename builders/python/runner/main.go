// The runner binary executes buildpacks for the Python language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	pythonappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/python/appengine/lib"
	pythonfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/python/functions_framework/lib"
	pythonfunctionsframeworkcompat "github.com/GoogleCloudPlatform/buildpacks/cmd/python/functions_framework_compat/lib"
	pythonlinkruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/python/link_runtime/lib"
	pythonmissingentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/python/missing_entrypoint/lib"
	pythonpip "github.com/GoogleCloudPlatform/buildpacks/cmd/python/pip/lib"
	pythonpoetry "github.com/GoogleCloudPlatform/buildpacks/cmd/python/poetry/lib"
	pythonruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/python/runtime/lib"
	pythonuv "github.com/GoogleCloudPlatform/buildpacks/cmd/python/uv/lib"
	pythonwebserver "github.com/GoogleCloudPlatform/buildpacks/cmd/python/webserver/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.python.appengine"] = gcp.BuildpackFuncs{
		Detect: pythonappengine.DetectFn,
		Build:  pythonappengine.BuildFn,
	}
	buildpacks["google.python.functions-framework"] = gcp.BuildpackFuncs{
		Detect: pythonfunctionsframework.DetectFn,
		Build:  pythonfunctionsframework.BuildFn,
	}
	buildpacks["google.python.functions-framework-compat"] = gcp.BuildpackFuncs{
		Detect: pythonfunctionsframeworkcompat.DetectFn,
		Build:  pythonfunctionsframeworkcompat.BuildFn,
	}
	buildpacks["google.python.link-runtime"] = gcp.BuildpackFuncs{
		Detect: pythonlinkruntime.DetectFn,
		Build:  pythonlinkruntime.BuildFn,
	}
	buildpacks["google.python.missing-entrypoint"] = gcp.BuildpackFuncs{
		Detect: pythonmissingentrypoint.DetectFn,
		Build:  pythonmissingentrypoint.BuildFn,
	}
	buildpacks["google.python.pip"] = gcp.BuildpackFuncs{
		Detect: pythonpip.DetectFn,
		Build:  pythonpip.BuildFn,
	}
	buildpacks["google.python.poetry"] = gcp.BuildpackFuncs{
		Detect: pythonpoetry.DetectFn,
		Build:  pythonpoetry.BuildFn,
	}
	buildpacks["google.python.runtime"] = gcp.BuildpackFuncs{
		Detect: pythonruntime.DetectFn,
		Build:  pythonruntime.BuildFn,
	}
	buildpacks["google.python.webserver"] = gcp.BuildpackFuncs{
		Detect: pythonwebserver.DetectFn,
		Build:  pythonwebserver.BuildFn,
	}
	buildpacks["google.python.uv"] = gcp.BuildpackFuncs{
		Detect: pythonuv.DetectFn,
		Build:  pythonuv.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/python/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
