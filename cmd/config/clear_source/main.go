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

// Implements config/clear_source buildpack.
// The clear_source buildpack clears the source out.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if devmode.Enabled(ctx) {
		ctx.OptOut("Keeping the source for dev mode")
	}

	if clearSource, ok := os.LookupEnv(env.ClearSource); ok {
		clear, err := strconv.ParseBool(clearSource)
		if err != nil {
			return fmt.Errorf("parsing %q: %v", env.ClearSource, err)
		}

		if clear {
			ctx.OptIn("%s set, clearing source", env.ClearSource)
		}
	}

	ctx.OptOut("%s not set, opting out", env.ClearSource)
	return nil
}

func buildFn(ctx *gcp.Context) error {
	ctx.Logf("Clearing source")

	defer func(now time.Time) {
		ctx.Span("Clear source", now, gcp.StatusOk)
	}(time.Now())

	for _, path := range ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "*")) {
		ctx.RemoveAll(path)
	}

	return nil
}
