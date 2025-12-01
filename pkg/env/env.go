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

// Package env specifies environment variables used to configure buildpack behavior.
package env

import (
	"fmt"
	"os"
	"strconv"
)

const (

	// Runtime is an env var used constrain autodetection in runtime buildpacks or to set runtime name in App Engine buildpacks.
	// Runtime must be respected by each runtime buildpack.
	// Example: `nodejs` will cause the nodejs/runtime buildpack to opt-in.
	Runtime = "GOOGLE_RUNTIME"

	// RuntimeVersion is an env var used to specify which runtime version to install.
	// RuntimeVersion must be respected by each runtime buildpack.
	// Example: `13.7.0` for Node.js, `1.14.1` for Go.
	RuntimeVersion = "GOOGLE_RUNTIME_VERSION"

	// DebugMode enables more verbose logging.
	// Example: `true`, `True`, `1` will enable development mode.
	DebugMode = "GOOGLE_DEBUG"

	// DevMode is an env var used to enable development mode in buildpacks.
	// DevMode should be respected by all buildpacks that are not product-specific.
	// Example: `true`, `True`, `1` will enable development mode.
	DevMode = "GOOGLE_DEVMODE"

	// Entrypoint is an env var used to override the default entrypoint.
	// Entrypoint should be respected by at least one buildpack in builders that are not product-specific.
	// Example: `gunicorn -p :8080 main:app` for Python.
	Entrypoint = "GOOGLE_ENTRYPOINT"

	// ClearSource is an env var used to clear source files from the final image.
	// Buildpacks for Go and Java support clearing the source.
	ClearSource = "GOOGLE_CLEAR_SOURCE"

	// Buildable is an env var used to specify the buildable unit to build.
	// Buildable should be respected by buildpacks that build source.
	// Example: `./maindir` for Go will build the package rooted at maindir.
	Buildable = "GOOGLE_BUILDABLE"

	// BuildArgs is an env var used to append arguments to the build command.
	// Example: `-Pprod` for Maven apps run "mvn clear package ... -Pprod" command.
	BuildArgs = "GOOGLE_BUILD_ARGS"

	// NoCache is an env var used to disable creation of cache layers.
	NoCache = "GOOGLE_NO_CACHE"

	// GAEMain is an env var used to specify path or fully qualified package name of the main package in App Engine buildpacks.
	// Behavior: In Go, the value is cleaned up and passed on to subsequent buildpacks as GOOGLE_BUILDABLE.
	GAEMain = "GAE_YAML_MAIN"

	// GaeApplicationYamlPath is set by gcloud for all GAE Flex runtimes. Flex java mvn deployment has
	// this env var too.
	GaeApplicationYamlPath = "GAE_APPLICATION_YAML_PATH"

	// AppEngineAPIs is an env var that enables access to App Engine APIs. Set to TRUE to enable.
	// Example: `true`, `True`, `1` will enable API access.
	AppEngineAPIs = "GAE_APP_ENGINE_APIS"

	// FunctionTarget is an env var used to specify function name.
	// FunctionTarget must be respected by all functions-framework buildpacks.
	// Example: `helloWorld` or any exported function name.
	FunctionTarget = "GOOGLE_FUNCTION_TARGET"
	// FunctionTargetLaunch is a launch time version of FunctionTarget.
	FunctionTargetLaunch = "FUNCTION_TARGET"

	// FunctionSource is an env var used to specify function source location.
	// FunctionSource must be respected by all functions-framework buildpacks.
	// Example: `./path/to/source` will build the function at the specfied path.
	FunctionSource = "GOOGLE_FUNCTION_SOURCE"
	// FunctionSourceLaunch is a launch time version of FunctionSource.
	FunctionSourceLaunch = "FUNCTION_SOURCE"

	// FunctionSignatureType is an env var used to specify function signature type.
	// FunctionSignatureType must be respected by all functions-framework buildpacks.
	// Example: `http` for HTTP-triggered functions or `event` for event-triggered functions.
	FunctionSignatureType = "GOOGLE_FUNCTION_SIGNATURE_TYPE"
	// FunctionSignatureTypeLaunch is a launch time version of FunctionSignatureType.
	FunctionSignatureTypeLaunch = "FUNCTION_SIGNATURE_TYPE"

	// GoGCFlags is an env var used to pass through compilation flags to the Go compiler.
	// Example: `-N -l` is used during debugging to disable optimizations and inlining.
	GoGCFlags = "GOOGLE_GOGCFLAGS"
	// GoLDFlags is an env var used to pass through linker flags to the Go linker.
	// Example: `-s -w` is sometimes used to strip and reduce binary size.
	GoLDFlags = "GOOGLE_GOLDFLAGS"

	// UseNativeImage is used to enable the GraalVM Java buildpack for native image compilation.
	// Example: `true`, `True`, `1` will enable development mode.
	UseNativeImage = "GOOGLE_JAVA_USE_NATIVE_IMAGE"

	// NativeImageBuildArgs is for additional build arguments to `native-image` when generating a GraalVM native image.
	// Example: `--enable-http --enable-https -H:ReflectionConfigurationFiles=native-image-config/picocli-reflect.json`
	NativeImageBuildArgs = "GOOGLE_JAVA_NATIVE_IMAGE_ARGS"

	// LabelPrefix is a prefix for values that will be added to the final
	// built user container. The prefix is stripped and the remainder forms the
	// label key. For example, "GOOGLE_LABEL_ABC=Some-Value" will result in a
	// label on the final container of "abc=Some-Value". The label key itself is
	// lowercased, underscores changed to dashes, and is prefixed with "google.".
	LabelPrefix = "GOOGLE_LABEL_"

	// ContainerMemoryHintMB is used to specify the amount of memory that will be allocated when running the container.
	ContainerMemoryHintMB = "GOOGLE_CONTAINER_MEMORY_HINT_MB"

	// XGoogleSkipRuntimeLaunch is used to enable an experimental builder feature to include the
	// runtime layer in the builder image and omit it from the launch image.
	XGoogleSkipRuntimeLaunch = "X_GOOGLE_SKIP_RUNTIME_LAUNCH"

	// XGoogleTargetPlatform is an envar used to specify the target platform for a build (gae, gcf or gcp).
	XGoogleTargetPlatform = "X_GOOGLE_TARGET_PLATFORM"

	// TargetPlatformAppEngine is the appengine value for 'X_GOOGLE_TARGET_PLATFORM'
	TargetPlatformAppEngine = "gae"

	// TargetPlatformFunctions is the functions value for 'X_GOOGLE_TARGET_PLATFORM'
	TargetPlatformFunctions = "gcf"

	// TargetPlatformFlex is the flex value for 'X_GOOGLE_TARGET_PLATFORM'
	TargetPlatformFlex = "flex"

	// TargetPlatformFAH is the firebase apphosting value for 'X_GOOGLE_TARGET_PLATFORM'
	TargetPlatformFAH = "fah"

	// FlexEnv is internal env variable to denote a flex application
	FlexEnv = "GOOGLE_FLEX_APPLICATION"

	// FlexMinVersion is the lowest version that is allowed to build.
	FlexMinVersion = "GOOGLE_FLEX_MIN_VERSION"

	// RuntimeImageRegion is the region to fetch runtime images.
	RuntimeImageRegion = "GOOGLE_RUNTIME_IMAGE_REGION"

	// FirebaseOutputDir is the directory to store the firebase output bundle.
	FirebaseOutputDir = "FIREBASE_OUTPUT_BUNDLE_DIR"

	// ServerlessRuntimesTarballs is an experiment flag to fetch tarballs from serverless-runtimes AR
	ServerlessRuntimesTarballs = "GOOGLE_USE_SERVERLESS_RUNTIMES_TARBALLS"

	// ColdStartImprovementsBuildStudy is an experiment flag to enable cold start improvements build study.
	ColdStartImprovementsBuildStudy = "EXPERIMENTAL_RUNTIMES_COLD_START_BUILD"

	// NodeCompileCache is an env var used to enable bytecode caching for Node.js applications.
	NodeCompileCache = "NODE_COMPILE_CACHE"

	// ReleaseTrack is an env var used to specify the release track for the Build.
	// Example: `ALPHA`, `BETA`, `GA`
	ReleaseTrack = "X_GOOGLE_RELEASE_TRACK"

	// BuildEnv is an env var used to specify the environment for the Build.
	// Example: dev, qual, prod.
	BuildEnv = "GOOGLE_BUILD_ENV"

	// BuildUniverse is an env var used to specify the universe for the Build.
	// Example: gdu, prp, tsq, tsp.
	BuildUniverse = "GOOGLE_BUILD_UNIVERSE"

	// PythonPackageManager is an env var used to specify the python package manager for the Build.
	// Example: `pip`, `uv`.
	PythonPackageManager = "GOOGLE_PYTHON_PACKAGE_MANAGER"
)

const (
	// ALPHA is the release track for alpha.
	ALPHA = "ALPHA"
	// BETA is the release track for beta.
	BETA = "BETA"
	// GA is the release track for GA.
	GA = "GA"
)

// IsAlphaSupported returns true if the release track is alpha.
func IsAlphaSupported() bool {
	return ALPHA == os.Getenv(ReleaseTrack)
}

// IsBetaSupported returns true if the release track is alpha or beta.
func IsBetaSupported() bool {
	return BETA == os.Getenv(ReleaseTrack) || IsAlphaSupported()
}

// IsGAE returns true if the buildpack target platform is gae.
func IsGAE() bool {
	return TargetPlatformAppEngine == os.Getenv(XGoogleTargetPlatform)
}

// IsFAH returns true if the buildpack target platform is fah.
func IsFAH() bool {
	return TargetPlatformFAH == os.Getenv(XGoogleTargetPlatform)
}

// IsGCP returns true if the buildpack target platform is not gae, gcf or flex.
func IsGCP() bool {
	return !IsGAE() && !IsGCF() && !IsFlex() && !IsFAH()
}

// IsGCF returns true if the buildpack target platform is gcf.
func IsGCF() bool {
	return TargetPlatformFunctions == os.Getenv(XGoogleTargetPlatform)
}

// IsFlex returns true if the buildpack target platform is flex
func IsFlex() bool {
	val, _ := IsPresentAndTrue(FlexEnv)
	return val || TargetPlatformFlex == os.Getenv(XGoogleTargetPlatform)
}

// IsDebugMode returns true if the buildpack debug mode is enabled.
func IsDebugMode() (bool, error) {
	return IsPresentAndTrue(DebugMode)
}

// IsDevMode indicates that the builder is running in Development mode.
func IsDevMode() (bool, error) {
	return IsPresentAndTrue(DevMode)
}

// IsUsingNativeImage returns true if the Java application should be built as a native image.
func IsUsingNativeImage() (bool, error) {
	return IsPresentAndTrue(UseNativeImage)
}

// IsPresentAndTrue returns true if the environment variable evaluates to True.
func IsPresentAndTrue(varName string) (bool, error) {
	varValue, present := os.LookupEnv(varName)
	if !present {
		return false, nil
	}

	parsed, err := strconv.ParseBool(varValue)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %v", varName, err)
	}

	return parsed, nil
}
