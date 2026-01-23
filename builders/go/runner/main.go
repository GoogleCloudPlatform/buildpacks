// The runner binary executes buildpacks for the Go language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	goappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/go/appengine/lib"
	goappenginegomod "github.com/GoogleCloudPlatform/buildpacks/cmd/go/appengine_gomod/lib"
	goappenginegopath "github.com/GoogleCloudPlatform/buildpacks/cmd/go/appengine_gopath/lib"
	gobuild "github.com/GoogleCloudPlatform/buildpacks/cmd/go/build/lib"
	goclearsource "github.com/GoogleCloudPlatform/buildpacks/cmd/go/clear_source/lib"
	goflexgomod "github.com/GoogleCloudPlatform/buildpacks/cmd/go/flex_gomod/lib"
	gofunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/go/functions_framework/lib"
	gogomod "github.com/GoogleCloudPlatform/buildpacks/cmd/go/gomod/lib"
	gogopath "github.com/GoogleCloudPlatform/buildpacks/cmd/go/gopath/lib"
	golegacyworker "github.com/GoogleCloudPlatform/buildpacks/cmd/go/legacy_worker/lib"
	goruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/go/runtime/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.go.appengine"] = gcp.BuildpackFuncs{
		Detect: goappengine.DetectFn,
		Build:  goappengine.BuildFn,
	}
	buildpacks["google.go.appengine-gomod"] = gcp.BuildpackFuncs{
		Detect: goappenginegomod.DetectFn,
		Build:  goappenginegomod.BuildFn,
	}
	buildpacks["google.go.flex-gomod"] = gcp.BuildpackFuncs{
		Detect: goflexgomod.DetectFn,
		Build:  goflexgomod.BuildFn,
	}
	buildpacks["google.go.appengine-gopath"] = gcp.BuildpackFuncs{
		Detect: goappenginegopath.DetectFn,
		Build:  goappenginegopath.BuildFn,
	}
	buildpacks["google.go.build"] = gcp.BuildpackFuncs{
		Detect: gobuild.DetectFn,
		Build:  gobuild.BuildFn,
	}
	buildpacks["google.go.clear-source"] = gcp.BuildpackFuncs{
		Detect: goclearsource.DetectFn,
		Build:  goclearsource.BuildFn,
	}
	buildpacks["google.go.functions-framework"] = gcp.BuildpackFuncs{
		Detect: gofunctionsframework.DetectFn,
		Build:  gofunctionsframework.BuildFn,
	}
	buildpacks["google.go.gomod"] = gcp.BuildpackFuncs{
		Detect: gogomod.DetectFn,
		Build:  gogomod.BuildFn,
	}
	buildpacks["google.go.gopath"] = gcp.BuildpackFuncs{
		Detect: gogopath.DetectFn,
		Build:  gogopath.BuildFn,
	}
	buildpacks["google.go.legacy-worker"] = gcp.BuildpackFuncs{
		Detect: golegacyworker.DetectFn,
		Build:  golegacyworker.BuildFn,
	}
	buildpacks["google.go.runtime"] = gcp.BuildpackFuncs{
		Detect: goruntime.DetectFn,
		Build:  goruntime.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/go/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
