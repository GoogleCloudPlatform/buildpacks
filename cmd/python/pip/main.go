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

// Implements python/pip buildpack.
// The pip buildpack installs dependencies using pip.
package main

import (
	"fmt"
	"os"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/python"
)

const (
	layerName = "pip"
)

// metadata represents metadata stored for a dependencies layer.
type metadata struct {
	PythonVersion   string `toml:"python_version"`
	DependencyHash  string `toml:"dependency_hash"`
	ExpiryTimestamp string `toml:"expiry_timestamp"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.FileExists("requirements.txt") {
		return gcp.OptOutFileNotFound("requirements.txt"), nil
	}
	return gcp.OptInFileFound("requirements.txt"), nil
}

func buildFn(ctx *gcp.Context) error {
	l := ctx.Layer(layerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayer)

	path, err := python.InstallRequirements(ctx, l, "requirements.txt")
	if err != nil {
		return fmt.Errorf("installing dependencies: %w", err)
	}

	ctx.Logf("Checking for incompatible dependencies.")
	result, err := ctx.ExecWithErr([]string{"python3", "-m", "pip", "check"}, gcp.WithEnv("PYTHONPATH="+path+":"+os.Getenv("PYTHONPATH")), gcp.WithUserAttribution)
	if result == nil {
		return fmt.Errorf("pip check: %w", err)
	}
	if result.ExitCode == 0 {
		return nil
	}
	// HACK: For backwards compatibility on App Engine and Cloud Functions Python 3.7 only report a warning.
	if strings.HasPrefix(python.Version(ctx), "Python 3.7") {
		ctx.Warnf("Found incompatible dependencies: %q", result.Stdout)
		return nil
	}
	return gcp.UserErrorf("found incompatible dependencies: %q", result.Stdout)

}
