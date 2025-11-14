// The runner binary executes buildpacks for the Universal builder.
package main

import (
	"flag"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/commonbuildpacks"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"

	// Buildpack libraries
	cppclearsource "github.com/GoogleCloudPlatform/buildpacks/cmd/cpp/clear_source/lib"
	cppfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/cpp/functions_framework/lib"
	dartcompile "github.com/GoogleCloudPlatform/buildpacks/cmd/dart/compile/lib"
	dartpub "github.com/GoogleCloudPlatform/buildpacks/cmd/dart/pub/lib"
	dartsdk "github.com/GoogleCloudPlatform/buildpacks/cmd/dart/sdk/lib"
	dotnetfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/functions_framework/lib"
	dotnetpublish "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/publish/lib"
	dotnetruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/runtime/lib"
	dotnetsdk "github.com/GoogleCloudPlatform/buildpacks/cmd/dotnet/sdk/lib"
	gobuild "github.com/GoogleCloudPlatform/buildpacks/cmd/go/build/lib"
	goclearsource "github.com/GoogleCloudPlatform/buildpacks/cmd/go/clear_source/lib"
	gofunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/go/functions_framework/lib"
	gogomod "github.com/GoogleCloudPlatform/buildpacks/cmd/go/gomod/lib"
	gogopath "github.com/GoogleCloudPlatform/buildpacks/cmd/go/gopath/lib"
	goruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/go/runtime/lib"
	javaclearsource "github.com/GoogleCloudPlatform/buildpacks/cmd/java/clear_source/lib"
	javaentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/java/entrypoint/lib"
	javaexplodedjar "github.com/GoogleCloudPlatform/buildpacks/cmd/java/exploded_jar/lib"
	javafunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/java/functions_framework/lib"
	javagraalvm "github.com/GoogleCloudPlatform/buildpacks/cmd/java/graalvm/lib"
	javagradle "github.com/GoogleCloudPlatform/buildpacks/cmd/java/gradle/lib"
	javamaven "github.com/GoogleCloudPlatform/buildpacks/cmd/java/maven/lib"
	javanativeimage "github.com/GoogleCloudPlatform/buildpacks/cmd/java/native_image/lib"
	javaruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/java/runtime/lib"
	javaspringboot "github.com/GoogleCloudPlatform/buildpacks/cmd/java/spring_boot/lib"
	nodejsfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/functions_framework/lib"
	nodejnpm "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/npm/lib"
	nodejppnpm "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/pnpm/lib"
	nodejruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/runtime/lib"
	nodejyarn "github.com/GoogleCloudPlatform/buildpacks/cmd/nodejs/yarn/lib"
	phpcomposer "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer/lib"
	phpcomposergcpbuild "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer_gcp_build/lib"
	phpcomposerinstall "github.com/GoogleCloudPlatform/buildpacks/cmd/php/composer_install/lib"
	phpruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/php/runtime/lib"
	phpwebconfig "github.com/GoogleCloudPlatform/buildpacks/cmd/php/webconfig/lib"
	pythonappengine "github.com/GoogleCloudPlatform/buildpacks/cmd/python/appengine/lib"
	pythonfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/python/functions_framework/lib"
	pythonmissingentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/python/missing_entrypoint/lib"
	pythonpip "github.com/GoogleCloudPlatform/buildpacks/cmd/python/pip/lib"
	pythonpoetry "github.com/GoogleCloudPlatform/buildpacks/cmd/python/poetry/lib"
	pythonruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/python/runtime/lib"
	pythonuv "github.com/GoogleCloudPlatform/buildpacks/cmd/python/uv/lib"
	pythonwebserver "github.com/GoogleCloudPlatform/buildpacks/cmd/python/webserver/lib"
	rubybundle "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/bundle/lib"
	rubyfunctionsframework "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/functions_framework/lib"
	rubymissingentrypoint "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/missing_entrypoint/lib"
	rubyrails "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/rails/lib"
	rubyrubygems "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/rubygems/lib"
	rubyruntime "github.com/GoogleCloudPlatform/buildpacks/cmd/ruby/runtime/lib"
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
	buildpacks["google.cpp.clear-source"] = gcp.BuildpackFuncs{
		Detect: cppclearsource.DetectFn,
		Build:  cppclearsource.BuildFn,
	}
	buildpacks["google.cpp.functions-framework"] = gcp.BuildpackFuncs{
		Detect: cppfunctionsframework.DetectFn,
		Build:  cppfunctionsframework.BuildFn,
	}
	buildpacks["google.dart.compile"] = gcp.BuildpackFuncs{
		Detect: dartcompile.DetectFn,
		Build:  dartcompile.BuildFn,
	}
	buildpacks["google.dart.pub"] = gcp.BuildpackFuncs{
		Detect: dartpub.DetectFn,
		Build:  dartpub.BuildFn,
	}
	buildpacks["google.dart.sdk"] = gcp.BuildpackFuncs{
		Detect: dartsdk.DetectFn,
		Build:  dartsdk.BuildFn,
	}
	buildpacks["google.dotnet.functions-framework"] = gcp.BuildpackFuncs{
		Detect: dotnetfunctionsframework.DetectFn,
		Build:  dotnetfunctionsframework.BuildFn,
	}
	buildpacks["google.dotnet.publish"] = gcp.BuildpackFuncs{
		Detect: dotnetpublish.DetectFn,
		Build:  dotnetpublish.BuildFn,
	}
	buildpacks["google.dotnet.runtime"] = gcp.BuildpackFuncs{
		Detect: dotnetruntime.DetectFn,
		Build:  dotnetruntime.BuildFn,
	}
	buildpacks["google.dotnet.sdk"] = gcp.BuildpackFuncs{
		Detect: dotnetsdk.DetectFn,
		Build:  dotnetsdk.BuildFn,
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
	buildpacks["google.go.runtime"] = gcp.BuildpackFuncs{
		Detect: goruntime.DetectFn,
		Build:  goruntime.BuildFn,
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
	buildpacks["google.java.graalvm"] = gcp.BuildpackFuncs{
		Detect: javagraalvm.DetectFn,
		Build:  javagraalvm.BuildFn,
	}
	buildpacks["google.java.gradle"] = gcp.BuildpackFuncs{
		Detect: javagradle.DetectFn,
		Build:  javagradle.BuildFn,
	}
	buildpacks["google.java.maven"] = gcp.BuildpackFuncs{
		Detect: javamaven.DetectFn,
		Build:  javamaven.BuildFn,
	}
	buildpacks["google.java.native-image"] = gcp.BuildpackFuncs{
		Detect: javanativeimage.DetectFn,
		Build:  javanativeimage.BuildFn,
	}
	buildpacks["google.java.runtime"] = gcp.BuildpackFuncs{
		Detect: javaruntime.DetectFn,
		Build:  javaruntime.BuildFn,
	}
	buildpacks["google.java.spring-boot"] = gcp.BuildpackFuncs{
		Detect: javaspringboot.DetectFn,
		Build:  javaspringboot.BuildFn,
	}
	buildpacks["google.nodejs.functions-framework"] = gcp.BuildpackFuncs{
		Detect: nodejsfunctionsframework.DetectFn,
		Build:  nodejsfunctionsframework.BuildFn,
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
	buildpacks["google.nodejs.yarn"] = gcp.BuildpackFuncs{
		Detect: nodejyarn.DetectFn,
		Build:  nodejyarn.BuildFn,
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
	buildpacks["google.php.runtime"] = gcp.BuildpackFuncs{
		Detect: phpruntime.DetectFn,
		Build:  phpruntime.BuildFn,
	}
	buildpacks["google.php.webconfig"] = gcp.BuildpackFuncs{
		Detect: phpwebconfig.DetectFn,
		Build:  phpwebconfig.BuildFn,
	}
	buildpacks["google.python.appengine"] = gcp.BuildpackFuncs{
		Detect: pythonappengine.DetectFn,
		Build:  pythonappengine.BuildFn,
	}
	buildpacks["google.python.functions-framework"] = gcp.BuildpackFuncs{
		Detect: pythonfunctionsframework.DetectFn,
		Build:  pythonfunctionsframework.BuildFn,
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
	buildpacks["google.ruby.bundle"] = gcp.BuildpackFuncs{
		Detect: rubybundle.DetectFn,
		Build:  rubybundle.BuildFn,
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
	buildpacks["google.ruby.rubygems"] = gcp.BuildpackFuncs{
		Detect: rubyrubygems.DetectFn,
		Build:  rubyrubygems.BuildFn,
	}
	buildpacks["google.ruby.runtime"] = gcp.BuildpackFuncs{
		Detect: rubyruntime.DetectFn,
		Build:  rubyruntime.BuildFn,
	}
	buildpacks["google.utils.nginx"] = gcp.BuildpackFuncs{
		Detect: utilsnginx.DetectFn,
		Build:  utilsnginx.BuildFn,
	}
}

// (-- LINT.ThenChange(//depot/google3/third_party/gcp_buildpacks/builders/gcp/base/runner/BUILD) --)

func main() {
	flag.Parse()
	gcp.MainRunner(buildpacks, buildpackID, phase)
}
