// The runner binary executes buildpacks for the Ruby language builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	nodejsruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/runtime/lib"
	nodejyarn "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/yarn/lib"
	rubyappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/appengine/lib"
	rubyappenginevalidation "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/appengine_validation/lib"
	rubybundle "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/bundle/lib"
	rubyflexentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/flex_entrypoint/lib"
	rubyfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/functions_framework/lib"
	rubymissingentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/missing_entrypoint/lib"
	rubyrails "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/rails/lib"
	rubyrubygems "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/rubygems/lib"
	rubyruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/runtime/lib"
)

var (
	buildpackID = flag.String("buildpack", "", "The ID of the buildpack to run (e.g., google.nodejs.runtime)")
	phase       = flag.String("phase", "", "The phase to run: 'detect' or 'build'")
)

// Register buildpack functions here
var buildpacks = commonbuildpacks.CommonBuildpacks()

// (-- LINT.IfChange --)
func init() {
	buildpacks["google.nodejs.runtime"] = gcp.BuildpackFuncs{
		Detect: nodejsruntime.DetectFn,
		Build:  nodejsruntime.BuildFn,
	}
	buildpacks["google.nodejs.yarn"] = gcp.BuildpackFuncs{
		Detect: nodejyarn.DetectFn,
		Build:  nodejyarn.BuildFn,
	}
	buildpacks["google.ruby.appengine"] = gcp.BuildpackFuncs{
		Detect: rubyappengine.DetectFn,
		Build:  rubyappengine.BuildFn,
	}
	buildpacks["google.ruby.appengine-validation"] = gcp.BuildpackFuncs{
		Detect: rubyappenginevalidation.DetectFn,
		Build:  rubyappenginevalidation.BuildFn,
	}
	buildpacks["google.ruby.bundle"] = gcp.BuildpackFuncs{
		Detect: rubybundle.DetectFn,
		Build:  rubybundle.BuildFn,
	}
	buildpacks["google.ruby.flex-entrypoint"] = gcp.BuildpackFuncs{
		Detect: rubyflexentrypoint.DetectFn,
		Build:  rubyflexentrypoint.BuildFn,
	}
	buildpacks["google.ruby.functions-framework"] = gcp.BuildpackFuncs{
		Detect: rubyfunctionsframework.DetectFn,
		Build:  rubyfunctionsframework.BuildFn,
	}
	buildpacks["google.ruby.missing-entrypoint"] = gcp.BuildpackFuncs{
		Detect: rubymissingentrypoint.DetectFn,
		Build:  rubymissingentrypoint.BuildFn,
	}
	buildpacks["google.ruby.rails"] = gcp.BuildpackFuncs{
		Detect: rubyrails.DetectFn,
		Build:  rubyrails.BuildFn,
	}
	buildpacks["google.ruby.runtime"] = gcp.BuildpackFuncs{
		Detect: rubyruntime.DetectFn,
		Build:  rubyruntime.BuildFn,
	}
	buildpacks["google.ruby.rubygems"] = gcp.BuildpackFuncs{
		Detect: rubyrubygems.DetectFn,
		Build:  rubyrubygems.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/ruby/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
