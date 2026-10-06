// Copyright 2023 Google LLC
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

package ruby

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	bundleIndicator  = "Gemfile.lock"
	bundle2Indicator = "gems.locked"
	railsIndicator   = "bin/rails"
	railsCommand     = "bin/rails server"
	rackIndicator    = "config.ru"
	rackCommand      = "rackup --port $PORT"
)

// InferEntrypoint is used to generate an entrypoint if it is missing in app.yaml
func InferEntrypoint(ctx *gcp.Context, srcDir string) (string, error) {
	indicatorCmds := map[string]string{
		railsIndicator: railsCommand,
		rackIndicator:  rackCommand,
	}
	for indc, cmd := range indicatorCmds {
		exists, err := ctx.FileExists(srcDir, indc)
		if err != nil {
			return "", err
		}
		if exists {
			return maybeBundle(ctx, srcDir, cmd)
		}
	}
	return "", gcp.UserErrorf("unable to infer entrypoint, please set the `entrypoint` field in app.yaml: https://cloud.google.com/appengine/docs/standard/ruby/runtime#application_startup")
}

func maybeBundle(ctx *gcp.Context, srcDir, cmd string) (string, error) {
	for _, indc := range []string{bundleIndicator, bundle2Indicator} {
		exists, err := ctx.FileExists(srcDir, indc)
		if err != nil {
			return "", err
		}
		if exists {
			return "bundle exec " + cmd, nil
		}
	}
	return cmd, nil
}
