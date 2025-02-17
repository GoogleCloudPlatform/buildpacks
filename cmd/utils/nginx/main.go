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

// Implements utils/nginx buildpack.
// The nginx buildpack installs the nginx web server, pid1 and serve binaries.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
)

const (
	// nginxVerConstraint is used to control updating to a new major version with any potential breaking change.
	// Update this to allow a new major version.
	nginxVerConstraint = "^1.21.6"

	// pid1VerConstraint is used to control updating to a new major version.
	pid1VerConstraint = "^1.0.0"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Always opt in.
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	// install nginx
	nl, err := install(ctx, "nginx", nginxVerConstraint, runtime.Nginx)
	if err != nil {
		return err
	}
	nl.LaunchEnvironment.Append("PATH", string(os.PathListSeparator), filepath.Join(nl.Path, "sbin"))
	nl.BuildEnvironment.Default("NGINX_ROOT", nl.Path)

	// install pid1
	pl, err := install(ctx, "pid1", pid1VerConstraint, runtime.Pid1)
	if err != nil {
		return err
	}

	pl.LaunchEnvironment.Append("PATH", string(os.PathListSeparator), pl.Path)
	pl.BuildEnvironment.Default("PID1_DIR", pl.Path)

	return nil
}

func install(ctx *gcp.Context, name, verConstraint string, ir runtime.InstallableRuntime) (*libcnb.Layer, error) {
	l, err := ctx.Layer(name, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}
	if _, err = runtime.InstallTarballIfNotCached(ctx, ir, verConstraint, l); err != nil {
		return nil, err
	}

	return l, nil
}
