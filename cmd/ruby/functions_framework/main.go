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
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/Masterminds/semver"
)

const (
	defaultSource = "app.rb"
	layerName     = "functions-framework"
)

var (
	// assumedVersion is the version of the framework used when we cannot determine a version.
	// To avoid breaking users on early versions of the framework when we could not easily
	// determine the framework version, this assumes users are on an early version.
	assumedVersion = semver.MustParse("0.2.0")
	// recommendedVersion is the lowest version for which a deprecation warning will be hidden.
	recommendedVersion = semver.MustParse("1.1.0")
	// validateTargetVersion is the minimum version that supports validating FUNCTION_TARGET.
	validateTargetVersion = semver.MustParse("0.7.0")
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
	l, err := ctx.Layer(layerName, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", layerName, err)
	}
	if err := ctx.SetFunctionsEnvVars(l); err != nil {
		return err
	}

	source, err := validateSource(ctx)
	if err != nil {
		return err
	}
	version, err := frameworkVersion(ctx)
	if err != nil {
		return err
	}
	if version.GreaterThan(validateTargetVersion) || version.Equal(validateTargetVersion) {
		if err := validateTarget(ctx, source); err != nil {
			return err
		}
	}
	if version.LessThan(recommendedVersion) {
		ctx.Warnf("Found a deprecated version of functions-framework (%s); consider updating your Gemfile to use functions_framework %s or later.", version, recommendedVersion)
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

	fnSourceExists, err := ctx.FileExists(fnSource)
	if err != nil {
		return "", err
	}
	if fnSourceExists {
		return fnSource, nil
	}
	if sourceEnvFound {
		return "", gcp.UserErrorf("%s specified file %q but it does not exist", env.FunctionSource, fnSource)
	}
	return "", gcp.UserErrorf("expected source file %q does not exist", fnSource)
}

// frameworkVersion validates framework installation and returns the major and minor components of its version
func frameworkVersion(ctx *gcp.Context) (*semver.Version, error) {
	cmd := []string{"bundle", "exec", "functions-framework-ruby", "--version"}
	result, err := ctx.Exec(cmd)
	// Failure to execute the binary at all implies the functions_framework is
	// not properly installed in the user's Gemfile.
	if result == nil || result.ExitCode == 127 {
		return nil, gcp.UserErrorf("unable to execute functions-framework-ruby; please ensure a recent version of the functions_framework gem is in your Gemfile")
	}
	// Frameworks older than 0.6 do not support the --version flag, signaled by a
	// nonzero error code. Respond with a pessimistic guess of the version.
	if err != nil {
		return assumedVersion, nil
	}
	version, perr := semver.NewVersion(result.Stdout)
	if perr != nil {
		return nil, gcp.UserErrorf(`failed to parse %q from "functions-framework-ruby --version": %v; please ensure a recent version of the functions_framework gem is in your Gemfile`, result.Stdout, perr)
	}
	return version, nil
}

// validateTarget validates that the given target is defined and can be executed
func validateTarget(ctx *gcp.Context, source string) error {
	target := os.Getenv(env.FunctionTarget)
	cmd := []string{"bundle", "exec", "functions-framework-ruby", "--quiet", "--verify", "--source", source, "--target", target}
	if fnSig, ok := os.LookupEnv(env.FunctionSignatureType); ok {
		cmd = append(cmd, "--signature-type", fnSig)
	}
	if result, err := ctx.Exec(cmd, gcp.WithEnv("MALLOC_ARENA_MAX=2", "LANG=C.utf8", "RACK_ENV=production"), gcp.WithUserAttribution); err != nil {
		return gcp.UserErrorf("failed to verify function target %q in source %q: %s", target, source, result.Stderr)
	}
	return nil
}
