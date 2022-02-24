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

// Implements ruby/appengine_validation buildpack.
// The appengine_validation buildpack ensures that Ruby version required by dependencies is not overly restrictive for runtime updates in App Engine.
package main

import (
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	gemfileExists, err := ctx.FileExists("Gemfile")
	if err != nil {
		return nil, err
	}
	if gemfileExists {
		return gcp.OptInFileFound("Gemfile"), nil
	}
	gemsRbExists, err := ctx.FileExists("gems.rb")
	if err != nil {
		return nil, err
	}
	if gemsRbExists {
		return gcp.OptInFileFound("gems.rb"), nil
	}
	return gcp.OptOut("no Gemfile or gems.rb found"), nil
}

func buildFn(ctx *gcp.Context) error {
	gemfile := ""
	gemfileExists, err := ctx.FileExists("Gemfile")
	if err != nil {
		return err
	}
	gemsRbExists, err := ctx.FileExists("gems.rb")
	if err != nil {
		return err
	}
	if gemfileExists {
		gemfile = "Gemfile"
		if gemsRbExists {
			ctx.Warnf("Gemfile and gems.gb both exist. Using Gemfile.")
		}
	} else if gemsRbExists {
		gemfile = "gems.rb"
	} else {
		return nil
	}
	script := filepath.Join(ctx.BuildpackRoot(), "scripts", "check_gemfile_version.rb")
	result, err := ctx.ExecWithErr([]string{"ruby", script, gemfile})
	if err != nil && result != nil && result.ExitCode != 0 {
		return gcp.UserErrorf(result.Stdout)
	}
	return nil
}
