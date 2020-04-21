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

// Implements php/composer_gcp_build buildpack.
// The composer_gcp_build buildpack runs the 'gcp-build' script in composer.json.
package main

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
)

const (
	cacheTag = "gcp-build dependencies"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.FileExists("composer.json") {
		ctx.OptOut("composer.json not found.")
	}

	p, err := php.ReadComposerJSON(ctx.ApplicationRoot())
	if err != nil {
		return fmt.Errorf("reading composer.json: %w", err)
	}
	if p.Scripts.GCPBuild == "" {
		ctx.OptOut("gcp-build script not found in composer.json.")
	}

	return nil
}

func buildFn(ctx *gcp.Context) error {
	_, err := php.ComposerInstall(ctx, cacheTag)
	if err != nil {
		return fmt.Errorf("composer install: %w", err)
	}

	ctx.ExecUser([]string{"composer", "run-script", "--timeout=600", "--no-dev", "gcp-build"})
	ctx.RemoveAll(php.Vendor)
	return nil
}
