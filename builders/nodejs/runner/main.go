// The runner binary executes buildpacks for the Nodejs language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	nodejsappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/appengine/lib"
	nodejsfirebaseangular "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/firebaseangular/lib"
	nodejsfirebasebundle "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/firebasebundle/lib"
	nodejfirebasenextjs "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/firebasenextjs/lib"
	nodejfirebasenx "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/firebasenx/lib"
	nodejsfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/functions_framework/lib"
	nodejslegacyworker "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/legacy_worker/lib"
	nodejnpm "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/npm/lib"
	nodejppnpm "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/pnpm/lib"
	nodejruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/runtime/lib"
	nodejsturborepo "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/turborepo/lib"
	nodejyarn "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/yarn/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.nodejs.appengine"] = gcp.BuildpackFuncs{
		Detect: nodejsappengine.DetectFn,
		Build:  nodejsappengine.BuildFn,
	}
	buildpacks["google.nodejs.firebaseangular"] = gcp.BuildpackFuncs{
		Detect: nodejsfirebaseangular.DetectFn,
		Build:  nodejsfirebaseangular.BuildFn,
	}
	buildpacks["google.nodejs.firebasebundle"] = gcp.BuildpackFuncs{
		Detect: nodejsfirebasebundle.DetectFn,
		Build:  nodejsfirebasebundle.BuildFn,
	}
	buildpacks["google.nodejs.firebasenextjs"] = gcp.BuildpackFuncs{
		Detect: nodejfirebasenextjs.DetectFn,
		Build:  nodejfirebasenextjs.BuildFn,
	}
	buildpacks["google.nodejs.firebasenx"] = gcp.BuildpackFuncs{
		Detect: nodejfirebasenx.DetectFn,
		Build:  nodejfirebasenx.BuildFn,
	}
	buildpacks["google.nodejs.functions-framework"] = gcp.BuildpackFuncs{
		Detect: nodejsfunctionsframework.DetectFn,
		Build:  nodejsfunctionsframework.BuildFn,
	}
	buildpacks["google.nodejs.legacy-worker"] = gcp.BuildpackFuncs{
		Detect: nodejslegacyworker.DetectFn,
		Build:  nodejslegacyworker.BuildFn,
	}
	buildpacks["google.nodejs.npm"] = gcp.BuildpackFuncs{
		Detect: nodejnpm.DetectFn,
		Build:  nodejnpm.BuildFn,
	}
	buildpacks["google.nodejs.pnpm"] = gcp.BuildpackFuncs{
		Detect: nodejppnpm.DetectFn,
		Build:  nodejppnpm.BuildFn,
	}
	buildpacks["google.nodejs.runtime"] = gcp.BuildpackFuncs{
		Detect: nodejruntime.DetectFn,
		Build:  nodejruntime.BuildFn,
	}
	buildpacks["google.nodejs.turborepo"] = gcp.BuildpackFuncs{
		Detect: nodejsturborepo.DetectFn,
		Build:  nodejsturborepo.BuildFn,
	}
	buildpacks["google.nodejs.yarn"] = gcp.BuildpackFuncs{
		Detect: nodejyarn.DetectFn,
		Build:  nodejyarn.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/nodejs/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
