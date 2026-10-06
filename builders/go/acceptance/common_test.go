// Copyright 2022 Google LLC
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

package acceptance_test

import (
	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	flex          = "google.config.flex"
	flexGoMod     = "google.go.flex-gomod"
	goBuild       = "google.go.build"
	goClearSource = "google.go.clear-source"
	goFF          = "google.go.functions-framework"
	goMod         = "google.go.gomod"
	goPath        = "google.go.gopath"
	goRuntime     = "google.go.runtime"
)
