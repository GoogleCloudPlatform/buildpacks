// The runner binary executes buildpacks for the PHP language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	phpappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/php/appengine/lib"
	phpcloudfunctions "github.com/GoogleCloudPlatform/buildpacks/cmd/php/cloudfunctions/lib"
	phpcomposer "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer/lib"
	phpcomposergcpbuild "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer_gcp_build/lib"
	phpcomposerinstall "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer_install/lib"
	phpfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/php/functions_framework/lib"
	phpruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/php/runtime/lib"
	phpsupervisor "github.com/GoogleCloudPlatform/buildpacks/cmd/php/supervisor/lib"
	phpwebconfig "github.com/GoogleCloudPlatform/buildpacks/cmd/php/webconfig/lib"
	pythonruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/python/runtime/lib"
	utilsnginx "github.com/GoogleCloudPlatform/buildpacks/cmd/utils/nginx/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.php.appengine"] = gcp.BuildpackFuncs{
		Detect: phpappengine.DetectFn,
		Build:  phpappengine.BuildFn,
	}
	buildpacks["google.php.cloudfunctions"] = gcp.BuildpackFuncs{
		Detect: phpcloudfunctions.DetectFn,
		Build:  phpcloudfunctions.BuildFn,
	}
	buildpacks["google.php.composer"] = gcp.BuildpackFuncs{
		Detect: phpcomposer.DetectFn,
		Build:  phpcomposer.BuildFn,
	}
	buildpacks["google.php.composer-gcp-build"] = gcp.BuildpackFuncs{
		Detect: phpcomposergcpbuild.DetectFn,
		Build:  phpcomposergcpbuild.BuildFn,
	}
	buildpacks["google.php.composer-install"] = gcp.BuildpackFuncs{
		Detect: phpcomposerinstall.DetectFn,
		Build:  phpcomposerinstall.BuildFn,
	}
	buildpacks["google.php.functions-framework"] = gcp.BuildpackFuncs{
		Detect: phpfunctionsframework.DetectFn,
		Build:  phpfunctionsframework.BuildFn,
	}
	buildpacks["google.php.runtime"] = gcp.BuildpackFuncs{
		Detect: phpruntime.DetectFn,
		Build:  phpruntime.BuildFn,
	}
	buildpacks["google.php.supervisor"] = gcp.BuildpackFuncs{
		Detect: phpsupervisor.DetectFn,
		Build:  phpsupervisor.BuildFn,
	}
	buildpacks["google.php.webconfig"] = gcp.BuildpackFuncs{
		Detect: phpwebconfig.DetectFn,
		Build:  phpwebconfig.BuildFn,
	}
	buildpacks["google.python.runtime"] = gcp.BuildpackFuncs{
		Detect: pythonruntime.DetectFn,
		Build:  pythonruntime.BuildFn,
	}
	buildpacks["google.utils.nginx"] = gcp.BuildpackFuncs{
		Detect: utilsnginx.DetectFn,
		Build:  utilsnginx.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/php/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
