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

// Package acceptance implements acceptance tests for a buildpack builder.
package acceptance

const (
	// Buildpack identifiers used to verify that buildpacks were or were not used.
	composer                = "google.php.composer"
	composerGCPBuild        = "google.php.composer-gcp-build"
	composerInstall         = "google.php.composer-install"
	cppFF                   = "google.cpp.functions-framework"
	dartCompile             = "google.dart.compile"
	dotnetFF                = "google.dotnet.functions-framework"
	dotnetPublish           = "google.dotnet.publish"
	dotnetRuntime           = "google.dotnet.runtime"
	dotnetSDK               = "google.dotnet.sdk"
	entrypoint              = "google.config.entrypoint"
	goBuild                 = "google.go.build"
	goClearSource           = "google.go.clear-source"
	goFF                    = "google.go.functions-framework"
	goMod                   = "google.go.gomod"
	goPath                  = "google.go.gopath"
	goRuntime               = "google.go.runtime"
	javaClearSource         = "google.java.clear-source"
	javaEntrypoint          = "google.java.entrypoint"
	javaExplodedJar         = "google.java.exploded-jar"
	javaFF                  = "google.java.functions-framework"
	javaGraalVM             = "google.java.graalvm"
	javaGradle              = "google.java.gradle"
	javaMaven               = "google.java.maven"
	javaNativeImage         = "google.java.native-image"
	javaRuntime             = "google.java.runtime"
	nodeFF                  = "google.nodejs.functions-framework"
	nodeNPM                 = "google.nodejs.npm"
	nodePNPM                = "google.nodejs.pnpm"
	nodeRuntime             = "google.nodejs.runtime"
	nodeYarn                = "google.nodejs.yarn"
	phpRuntime              = "google.php.runtime"
	phpWebConfig            = "google.php.webconfig"
	pythonFF                = "google.python.functions-framework"
	pythonPIP               = "google.python.pip"
	pythonRuntime           = "google.python.runtime"
	pythonMissingEntrypoint = "google.python.missing-entrypoint"
	pythonWebserver         = "google.python.webserver"
	pythonPoetry            = "google.python.poetry"
	pythonUV                = "google.python.uv"
	rubyBundle              = "google.ruby.bundle"
	rubyFF                  = "google.ruby.functions-framework"
	rubyRails               = "google.ruby.rails"
	rubyRuntime             = "google.ruby.runtime"
	utilsNginx              = "google.utils.nginx"
)
