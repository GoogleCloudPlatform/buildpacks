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

// Implements php/composer buildpack.
// The composer buildpack installs dependencies using composer.
package main

import (
	"fmt"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
)

const (
	cacheTag = "prod dependencies"
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
	return gcp.OptInFileFound("composer.json"), nil
}

func buildFn(ctx *gcp.Context) error {
	_, err := php.ComposerInstall(ctx, cacheTag)
	if err != nil {
		return fmt.Errorf("composer install: %w", err)
	}

	return nil
}
