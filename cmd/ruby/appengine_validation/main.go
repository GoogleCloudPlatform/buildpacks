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

// Implements /bin/build for ruby/appengine_validation buildpack.
package main

import (
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if ctx.FileExists("Gemfile") || ctx.FileExists("gems.rb") {
		return nil
	}
	ctx.OptOut("No Gemfile nor gems.rb found.")
	return nil
}

func buildFn(ctx *gcp.Context) error {
	gemfile := ""
	if ctx.FileExists("Gemfile") {
		gemfile = "Gemfile"
		if ctx.FileExists("gems.rb") {
			ctx.Warnf("Gemfile and gems.gb both exist. Using Gemfile.")
		}
	} else if ctx.FileExists("gems.rb") {
		gemfile = "gems.rb"
	}
	if gemfile == "" {
		return nil
	}

	script := filepath.Join(ctx.BuildpackRoot(), "scripts", "check_gemfile_version.rb")
	cmd := []string{"ruby", script, gemfile}
	result, err := ctx.ExecWithErr(cmd)
	if err != nil && result != nil && result.ExitCode != 0 {
		return gcp.UserErrorf(result.Stdout)
	}
	return err
}
