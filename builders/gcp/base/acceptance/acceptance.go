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
	entrypoint      = "google.config.entrypoint"
	cppFF           = "google.cpp.functions-framework"
	dartCompile     = "google.dart.compile"
	dotnetFF        = "google.dotnet.functions-framework"
	dotnetPublish   = "google.dotnet.publish"
	dotnetRuntime   = "google.dotnet.runtime"
	goBuild         = "google.go.build"
	goClearSource   = "google.go.clear_source"
	goFF            = "google.go.functions-framework"
	goMod           = "google.go.gomod"
	goPath          = "google.go.gopath"
	goRuntime       = "google.go.runtime"
	javaClearSource = "google.java.clear_source"
	javaEntrypoint  = "google.java.entrypoint"
	javaExplodedJar = "google.java.exploded-jar"
	javaFF          = "google.java.functions-framework"
	javaGradle      = "google.java.gradle"
	javaGraalVM     = "google.java.graalvm"
	javaMaven       = "google.java.maven"
	javaNativeImage = "google.java.native-image"
	javaRuntime     = "google.java.runtime"
	nodeFF          = "google.nodejs.functions-framework"
	nodeNPM         = "google.nodejs.npm"
	nodeRuntime     = "google.nodejs.runtime"
	nodeYarn        = "google.nodejs.yarn"
	pythonFF        = "google.python.functions-framework"
	pythonPIP       = "google.python.pip"
	pythonRuntime   = "google.python.runtime"
	rubyRuntime     = "google.ruby.runtime"
	rubyBundle      = "google.ruby.bundle"
	rubyRails       = "google.ruby.rails"
)
