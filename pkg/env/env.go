# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from typing import Optional

def _is_present_and_true(env_var: str) -> tuple[bool, Optional[Exception]]:
    value = os.getenv(env_var)
    if value is None:
        return False, None
    
    if value.lower() in {'true', '1'}:
        return True, None
    elif value.lower() in {'false', '0'}:
        return False, None
    else:
        return False, ValueError(f"Invalid boolean value for {env_var}: {value}")

# Runtime is an env var used to constrain autodetection in runtime buildpacks or to set runtime name in App Engine buildpacks.
Runtime = "GOOGLE_RUNTIME"

# RuntimeVersion is an env var used to specify which runtime version to install.
RuntimeVersion = "GOOGLE_RUNTIME_VERSION"

# DebugMode enables more verbose logging.
DebugMode = "GOOGLE_DEBUG"

# DevMode is an env var used to enable development mode in buildpacks.
DevMode = "GOOGLE_DEVMODE"

# DevSync is an env var used to enable development sync mode in buildpacks.
DevSync = "GOOGLE_DEVSYNC"

# DevSyncInitEntrypoint is an env var used to specify the initial entrypoint for dev sync mode.
DevSyncInitEntrypoint = "GOOGLE_DEV_SYNC_INIT_ENTRYPOINT"

# XGoogleDevSyncUseRunitUniversalMaker is an experiment flag to enable Runit supervision and the new universal_maker for DevSync on Ubuntu 24.04.
XGoogleDevSyncUseRunitUniversalMaker = "X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER"

# XGoogleDevSyncActivated is an experiment flag to enable DevSync logic in buildpacks.
XGoogleDevSyncActivated = "X_GOOGLE_DEVSYNC_ACTIVATED"

# Entrypoint is an env var used to override the default entrypoint.
Entrypoint = "GOOGLE_ENTRYPOINT"

# ClearSource is an env var used to clear source files from the final image.
ClearSource = "GOOGLE_CLEAR_SOURCE"

# Buildable is an env var used to specify the buildable unit to build.
Buildable = "GOOGLE_BUILDABLE"

# BuildArgs is an env var used to append arguments to the build command.
BuildArgs = "GOOGLE_BUILD_ARGS"

# NoCache is an env var used to disable creation of cache layers.
NoCache = "GOOGLE_NO_CACHE"

# GAEMain is an env var used to specify path or fully qualified package name of the main package in App Engine buildpacks.
GAEMain = "GAE_YAML_MAIN"

# GaeApplicationYamlPath is set by gcloud for all GAE Flex runtimes. Flex java mvn deployment has
# this env var too.
GaeApplicationYamlPath = "GAE_APPLICATION_YAML_PATH"

# AppEngineAPIs is an env var that enables access to App Engine APIs. Set to TRUE to enable.
AppEngineAPIs = "GAE_APP_ENGINE_APIS"

# FunctionTarget is an env var used to specify function name.
FunctionTarget = "GOOGLE_FUNCTION_TARGET"
FunctionTargetLaunch = "FUNCTION_TARGET"

# FunctionSource is an env var used to specify function source location.
FunctionSource = "GOOGLE_FUNCTION_SOURCE"
FunctionSourceLaunch = "FUNCTION_SOURCE"

# FunctionSignatureType is an env var used to specify function signature type.
FunctionSignatureType = "GOOGLE_FUNCTION_SIGNATURE_TYPE"
FunctionSignatureTypeLaunch = "FUNCTION_SIGNATURE_TYPE"

# GoGCFlags is an env var used to pass through compilation flags to the Go compiler.
GoGCFlags = "GOOGLE_GOGCFLAGS"

# GoLDFlags is an env var used to pass through linker flags to the Go linker.
GoLDFlags = "GOOGLE_GOLDFLAGS"

# UseNativeImage is used to enable the GraalVM Java buildpack for native image compilation.
UseNativeImage = "GOOGLE_JAVA_USE_NATIVE_IMAGE"

# NativeImageBuildArgs is for additional build arguments to `native-image` when generating a GraalVM native image.
NativeImageBuildArgs = "GOOGLE_JAVA_NATIVE_IMAGE_ARGS"

# LabelPrefix is a prefix for values that will be added to the final
# built user container. The prefix is stripped and the remainder forms the
# label key. For example, "GOOGLE_LABEL_ABC=Some-Value" will result in a
# label on the final container of "abc=Some-Value". The label key itself is
# lowercased, underscores changed to dashes, and is prefixed with "google.".
LabelPrefix = "GOOGLE_LABEL_"

# ContainerMemoryHintMB is used to specify the amount of memory that will be allocated when running the container.
ContainerMemoryHintMB = "GOOGLE_CONTAINER_MEMORY_HINT_MB"

# XGoogleSkipRuntimeLaunch is used to enable an experimental builder feature to include the
# runtime layer in the builder image and omit it from the launch image.
XGoogleSkipRuntimeLaunch = "X_GOOGLE_SKIP_RUNTIME_LAUNCH"

# XGoogleTargetPlatform is an envar used to specify the target platform for a build (gae, gcf or gcp).
XGoogleTargetPlatform = "X_GOOGLE_TARGET_PLATFORM"

# TargetPlatformAppEngine is the appengine value for 'X_GOOGLE_TARGET_PLATFORM'
TargetPlatformAppEngine = "gae"

# TargetPlatformFunctions is the functions value for 'X_GOOGLE_TARGET_PLATFORM'
TargetPlatformFunctions = "gcf"

# TargetPlatformFlex is the flex value for 'X_GOOGLE_TARGET_PLATFORM'
TargetPlatformFlex = "flex"

# TargetPlatformFAH is the firebase apphosting value for 'X_GOOGLE_TARGET_PLATFORM'
TargetPlatformFAH = "fah"

# FlexEnv is internal env variable to denote a flex application
FlexEnv = "GOOGLE_FLEX_APPLICATION"

# FlexMinVersion is the lowest version that is allowed to build.
FlexMinVersion = "GOOGLE_FLEX_MIN_VERSION"

# RuntimeImageRegion is the region to fetch runtime images.
RuntimeImageRegion = "GOOGLE_RUNTIME_IMAGE_REGION"

# FirebaseOutputDir is the directory to store the firebase output bundle.
FirebaseOutputDir = "FIREBASE_OUTPUT_BUNDLE_DIR"

# ServerlessRuntimesTarballs is an experiment flag to fetch tarballs from serverless-runtimes AR
ServerlessRuntimesTarballs = "GOOGLE_USE_SERVERLESS_RUNTIMES_TARBALLS"

# ColdStartImprovementsBuildStudy is an experiment flag to enable cold start improvements build study.
ColdStartImprovementsBuildStudy = "EXPERIMENTAL_RUNTIMES_COLD_START_BUILD"

# FasterLanguageTarballInstallation is an experiment flag to enable faster language tarball installation.
FasterLanguageTarballInstallation = "X_GOOGLE_FASTER_LANGUAGE_TARBALL_INSTALLATION"

# FasterTarballExtraction is an experiment flag to enable faster tarball extraction.
FasterTarballExtraction = "X_GOOGLE_USE_ZSTD_FOR_EXTRACTION"

# NodeCompileCache is an env var used to enable bytecode caching for Node.js applications.
NodeCompileCache = "NODE_COMPILE_CACHE"

# ReleaseTrack is an env var used to specify the release track for the Build.
ReleaseTrack = "X_GOOGLE_RELEASE_TRACK"

# BuildEnv is an env var used to specify the environment for the Build.
BuildEnv = "GOOGLE_BUILD_ENV"

# BuildUniverse is an env var used to specify the universe for the Build.
BuildUniverse = "GOOGLE_BUILD_UNIVERSE"

# TPCTarballProject is an env var used to specify the project for the TPC tarball.
TPCTarballProject = "GOOGLE_TPC_TARBALL_PROJECT"

# TPCHostname is an env var used to specify the hostname for the TPC build.
TPCHostname = "GOOGLE_TPC_HOSTNAME"

# PythonPackageManager is an env var used to specify the python package manager for the Build.
PythonPackageManager = "GOOGLE_PYTHON_PACKAGE_MANAGER"

# AllowVulnerableDependencies is an env var used to disable react2shell vulnerability checks.
AllowVulnerableDependencies = "GOOGLE_ALLOW_VULNERABLE_DEPENDENCIES"

# GoogleUseGenericFirebaseBundle enables the generic firebase bundle buildpack.
GoogleUseGenericFirebaseBundle = "GOOGLE_USE_GENERIC_FIREBASEBUNDLE"

# PackageManager is an env var used to specify the package manager for the Build.
PackageManager = "GOOGLE_PACKAGE_MANAGER"

# PipTargetDir is the environment variable used to specify the target directory for pip
# installation for the maker use case.
PipTargetDir = "GOOGLE_PIP_TARGET_DIR"

# StaticServe indicates that the static serve buildpack was invoked.
StaticServe = "GOOGLE_STATIC_SERVE"

# BuildIntelligenceFeature is an experiment flag to enable build intelligence feature.
BuildIntelligenceFeature = "X_GOOGLE_BUILD_INTELLIGENCE"


ALPHA = "ALPHA"
BETA = "BETA"
GA = "GA"

def is_alpha_supported() -> tuple[bool, Optional[Exception]]:
    release_track = os.getenv(ReleaseTrack)
    return (release_track == ALPHA), None

def is_beta_supported() -> tuple[bool, Optional[Exception]]:
    release_track = os.getenv(ReleaseTrack)
    return (release_track in {ALPHA, BETA}), None

def is_gae() -> bool:
    target_platform = os.getenv(XGoogleTargetPlatform)
    return target_platform == TargetPlatformAppEngine

def is_fah() -> bool:
    target_platform = os.getenv(XGoogleTargetPlatform)
    return target_platform == TargetPlatformFAH

def is_gcf() -> bool:
    target_platform = os.getenv(XGoogleTargetPlatform)
    return target_platform == TargetPlatformFunctions

def is_flex() -> bool:
    dev_mode, _ = _is_present_and_true(FlexEnv)
    target_platform = os.getenv(XGoogleTargetPlatform)
    return dev_mode or (target_platform == TargetPlatformFlex)

def is_gcp() -> bool:
    return not (is_gae() or is_gcf() or is_flex() or is_fah())

def is_debug_mode() -> tuple[bool, Optional[Exception]]:
    return _is_present_and_true(DebugMode)

def is_dev_mode() -> tuple[bool, Optional[Exception]]:
    return _is_present_and_true(DevMode)

def is_dev_sync() -> tuple[bool, Optional[Exception]]:
    activated_set, err = _is_present_and_true(XGoogleDevSyncActivated)
    if not activated_set or err:
        return False, err
    
    dev_sync_set, err = _is_present_and_true(DevSync)
    return dev_sync_set, err

def is_dev_sync_use_runit_universal_maker() -> tuple[bool, Optional[Exception]]:
    return _is_present_and_true(XGoogleDevSyncUseRunitUniversalMaker)

def is_using_native_image() -> tuple[bool, Optional[Exception]]:
    return _is_present_and_true(UseNativeImage)

def using_static_serve() -> tuple[bool, Optional[Exception]]:
    return _is_present_and_true(StaticServe)

def is_static_base_image() -> bool:
    runtime = os.getenv(Runtime)
    if not runtime:
        return False
    return runtime.startswith("static")
