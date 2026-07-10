// Copyright 2025 Google LLC
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
// The nginx buildpack installs the nginx web server and pid1 binaries.
package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/static"
)

const (
	// defaultNginxVerConstraint is used to control updating to a new major version with any potential breaking change.
	// Update this to allow a new major version.
	defaultNginxVerConstraint = "^1.21.6"
	// pid1VerConstraint is used to control updating to a new major version.
	pid1VerConstraint = "^1.0.0"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// Always opt in.
	return gcp.OptInAlways(), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	usingStaticServe, err := env.UsingStaticServe()
	if err != nil {
		ctx.Warnf("failed to parse GOOGLE_STATIC_SERVE: %v, defaulting to false", err)
	}

	nginxVerConstraint := defaultNginxVerConstraint

	if usingStaticServe {
		runtimeName := os.Getenv(env.Runtime)
		nginxVerConstraint = static.NginxVersionConstraint(runtimeName)
	}

	// install nginx
	nl, err := install(ctx, "nginx", nginxVerConstraint, runtime.Nginx)
	if err != nil {
		return err
	}
	nl.LaunchEnvironment.Append("PATH", string(os.PathListSeparator), filepath.Join(nl.Path, "sbin"))
	nl.BuildEnvironment.Default("NGINX_ROOT", nl.Path)

	// Install pid1 unless the static serve buildpack has marked the build environment to exclude it.
	if !usingStaticServe {
		// install pid1
		pl, err := install(ctx, "pid1", pid1VerConstraint, runtime.Pid1)
		if err != nil {
			return err
		}

		pl.LaunchEnvironment.Append("PATH", string(os.PathListSeparator), pl.Path)
		pl.BuildEnvironment.Default("PID1_DIR", pl.Path)
	}

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
