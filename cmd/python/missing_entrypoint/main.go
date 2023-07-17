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

// Implements python/missing-entrypoint buildpack.
// This buildpack's goal is to display a clear error message when
// no entrypoint is defined on a Python application.
package main

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("python"); result != nil {
		return result, nil
	}

	atLeastOne, err := ctx.HasAtLeastOne("*.py")
	if err != nil {
		return nil, fmt.Errorf("finding *.py files: %w", err)
	}
	if !atLeastOne {
		return gcp.OptOut("no .py files found"), nil
	}
	return gcp.OptIn("found .py files"), nil
}

func buildFn(ctx *gcp.Context) error {
	hasMain, err := ctx.HasAtLeastOne("main.py")
	if err != nil {
		return fmt.Errorf("finding main.py files: %w", err)
	}
	if !hasMain {
		return fmt.Errorf("for Python, provide a main.py file or set an entrypoint with %q env var or by creating a %q file", env.Entrypoint, "Procfile")
	}

	cmd := []string{"gunicorn", "-b", ":8080", "main:app"}
	ctx.Logf("Setting default entrypoint: %q", strings.Join(cmd, " "))
	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}
