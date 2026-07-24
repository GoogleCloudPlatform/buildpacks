// Copyright 2021 Google LLC
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

package nodejs_test

import (
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb/v2"
)

func ExampleMakerYarnInstaller() {
	ctx := gcpbuildpack.NewContext()
	var yarnLayer *libcnb.Layer
	var pjs *nodejs.PackageJSON

	type yarnInstaller interface {
		InstallYarn(ctx *gcpbuildpack.Context, yarnLayer *libcnb.Layer, pjs *nodejs.PackageJSON) error
	}

	if cap := ctx.Capability(nodejs.YarnInstallerCapability); cap != nil {
		_ = cap.(yarnInstaller).InstallYarn(ctx, yarnLayer, pjs)
	}

	// Output:
}

func ExampleMakerYarn1ModuleInstaller() {
	ctx := gcpbuildpack.NewContext()
	var pjs *nodejs.PackageJSON

	type moduleInstaller interface {
		InstallModules(ctx *gcpbuildpack.Context, pjs *nodejs.PackageJSON) error
	}

	if cap := ctx.Capability(nodejs.Yarn1ModuleInstallerCapability); cap != nil {
		_ = cap.(moduleInstaller).InstallModules(ctx, pjs)
	}

	// Output:
}
