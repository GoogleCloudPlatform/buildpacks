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
	entrypoint     = "google.config.entrypoint"
	clearSource    = "google.config.clear_source"
	dotnetPublish  = "google.dotnet.publish"
	dotnetRuntime  = "google.dotnet.runtime"
	goBuild        = "google.go.build"
	goFF           = "google.go.functions-framework"
	goPath         = "google.go.gopath"
	goRuntime      = "google.go.runtime"
	javaEntrypoint = "google.java.entrypoint"
	javaGradle     = "google.java.gradle"
	javaMaven      = "google.java.maven"
	javaRuntime    = "google.java.runtime"
	nodeFF         = "google.nodejs.functions-framework"
	nodeNPM        = "google.nodejs.npm"
	nodeRuntime    = "google.nodejs.runtime"
	nodeYarn       = "google.nodejs.yarn"
	pythonFF       = "google.python.functions-framework"
	pythonPIP      = "google.python.pip"
	pythonRuntime  = "google.python.runtime"
)
