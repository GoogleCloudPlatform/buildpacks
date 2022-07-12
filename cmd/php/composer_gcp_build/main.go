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

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	composerJSONExists, err := ctx.FileExists("composer.json")
	if err != nil {
		return nil, err
	}
	if !composerJSONExists {
		return gcp.OptOutFileNotFound("composer.json"), nil
	}

	p, err := php.ReadComposerJSON(ctx.ApplicationRoot())
	if err != nil {
		return nil, fmt.Errorf("reading composer.json: %w", err)
	}
	if p.Scripts.GCPBuild == "" {
		return gcp.OptOut("gcp-build script not found in composer.json"), nil
	}

	return gcp.OptIn("found composer.json with a gcp-build script"), nil
}

func buildFn(ctx *gcp.Context) error {
	_, err := php.ComposerInstall(ctx, cacheTag)
	if err != nil {
		return fmt.Errorf("composer install: %w", err)
	}

	if _, err := ctx.ExecWithErr([]string{"composer", "run-script", "--timeout=600", "--no-dev", "gcp-build"}, gcp.WithUserAttribution); err != nil {
		return err
	}
	if err := ctx.RemoveAll(php.Vendor); err != nil {
		return err
	}
	return nil
}
