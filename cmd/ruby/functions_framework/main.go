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

// Implements ruby/functions_framework buildpack.
// The functions_framework buildpack sets up the execution environment to
// run the Ruby Functions Framework. The framework itself, with its converter,
// is always installed as a dependency.
package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	defaultSource  = "app.rb"
	layerName      = "functions-framework"
	recommendMajor = 0
	recommendMinor = 7
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, ok := os.LookupEnv(env.FunctionTarget); ok {
		return gcp.OptInEnvSet(env.FunctionTarget), nil
	}
	return gcp.OptOutEnvNotSet(env.FunctionTarget), nil
}

func buildFn(ctx *gcp.Context) error {
	// The framework has been installed with the dependencies, so this layer is
	// used only for env vars.
	l := ctx.Layer(layerName, gcp.LaunchLayer)
	ctx.SetFunctionsEnvVars(l)

	source, err := validateSource(ctx)
	if err != nil {
		return err
	}
	major, minor, err := frameworkVersion(ctx)
	if err != nil {
		return err
	}
	// Target validation is available in framework 0.7 or later.
	if major > 0 || minor >= 7 {
		if err := validateTarget(ctx, source); err != nil {
			return err
		}
	}
	if major < recommendMajor || major == recommendMajor && minor < recommendMinor {
		ctx.Warnf("a deprecated version of functions_framework is in use; consider updating your Gemfile to use functions_framework %d.%d or later.", recommendMajor, recommendMinor)
	}

	ctx.AddWebProcess([]string{"bundle", "exec", "functions-framework-ruby"})

	return nil
}

// validateSource validates the existence of and returns the source file
func validateSource(ctx *gcp.Context) (string, error) {
	fnSource, sourceEnvFound := os.LookupEnv(env.FunctionSource)
	if !sourceEnvFound {
		fnSource = defaultSource
	}

	if ctx.FileExists(fnSource) {
		return fnSource, nil
	}
	if sourceEnvFound {
		return "", gcp.UserErrorf("%s specified file %q but it does not exist", env.FunctionSource, fnSource)
	}
	return "", gcp.UserErrorf("expected source file %q does not exist", fnSource)
}

// frameworkVersion validates framework installation and returns the major and minor components of its version
func frameworkVersion(ctx *gcp.Context) (int, int, error) {
	cmd := []string{"bundle", "exec", "functions-framework-ruby", "--version"}
	result, err := ctx.ExecWithErr(cmd)
	// Failure to execute the binary at all implies the functions_framework is
	// not properly installed in the user's Gemfile.
	if result == nil || result.ExitCode == 127 {
		return 0, 0, gcp.UserErrorf("unable to execute functions-framework-ruby; please ensure a recent version of the functions_framework gem is in your Gemfile")
	}
	// Frameworks older than 0.6 do not support the --version flag, signaled by a
	// nonzero error code. Respond with a pessimistic guess of the version.
	if err != nil {
		return 0, 2, nil
	}
	vsegs := strings.Split(result.Stdout, ".")
	if len(vsegs) == 3 {
		if major, err := strconv.Atoi(vsegs[0]); err == nil {
			if minor, err := strconv.Atoi(vsegs[1]); err == nil {
				return major, minor, nil
			}
		}
	}
	return 0, 0, gcp.UserErrorf("unexpected output %q from \"functions-framework-ruby --version\"; please ensure a recent version of the functions_framework gem is in your Gemfile", result.Stdout)
}

// validateTarget validates that the given target is defined and can be executed
func validateTarget(ctx *gcp.Context, source string) error {
	target := os.Getenv(env.FunctionTarget)
	cmd := []string{"bundle", "exec", "functions-framework-ruby", "--quiet", "--verify", "--source", source, "--target", target}
	if fnSig, ok := os.LookupEnv(env.FunctionSignatureType); ok {
		cmd = append(cmd, "--signature-type", fnSig)
	}
	if result, err := ctx.ExecWithErr(cmd, gcp.WithUserAttribution); err != nil {
		return gcp.UserErrorf("failed to verify function target %q in source %q: %s", target, source, result.Stderr)
	}
	return nil
}
