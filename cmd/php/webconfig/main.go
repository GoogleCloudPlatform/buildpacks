// Copyright 2022 Google LLC
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

// Implements php/webconfig buildpack.
// The runtime buildpack installs the config needed for PHP runtime.
package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/php"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/webconfig"
	"github.com/Masterminds/semver"
)

const (
	// pid1
	appSocket = "app.sock"
	pid1Log   = "pid1.log"

	defaultFlexAddress = "127.0.0.1:9000"

	// nginx
	defaultFrontController = "index.php"
	defaultNginxBinary     = "nginx"
	defaultNginxPort       = 8080
	defaultRoot            = "/workspace"
	nginxConf              = "nginx.conf"
	nginxLog               = "nginx.log"

	// php-fpm
	defaultDynamicWorkers = false
	defaultFPMBinary      = "php-fpm"
	defaultFPMWorkers     = 2
	phpFpmPid             = "php-fpm.pid"
)

var (
	overrides = webconfig.OverrideProperties{}
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
}

func buildFn(ctx *gcp.Context) error {
	// create webconfig layer
	l, err := ctx.Layer("webconfig", gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	if env.IsFlex() {
		runtimeConfig, err := appyaml.PhpConfiguration(ctx.ApplicationRoot())
		if err != nil {
			return err
		}
		overrides = webconfig.OverriddenProperties(ctx, runtimeConfig)
		webconfig.SetEnvVariables(l, overrides)
	}

	if customNginxConf, present := os.LookupEnv(php.CustomNginxConfig); present {
		overrides.NginxConfOverride = true
		overrides.NginxConfOverrideFileName = filepath.Join(defaultRoot, customNginxConf)
	}

	fpmConfFile, err := writeFpmConfig(ctx, l.Path, overrides)
	if err != nil {
		return err
	}
	defer fpmConfFile.Close()

	nginxServerConfFile, err := writeNginxServerConfig(l.Path, overrides)
	if err != nil {
		return err
	}
	defer nginxServerConfFile.Close()

	procExists, err := ctx.FileExists("Procfile")
	if err != nil {
		return err
	}
	_, entrypointExists := os.LookupEnv(env.Entrypoint)

	if !procExists && !entrypointExists {
		cmd := []string{
			filepath.Join(os.Getenv("PID1_DIR"), "pid1"),
			"--nginxBinaryPath", defaultNginxBinary,
			"--nginxErrLogFilePath", filepath.Join(l.Path, nginxLog),
			"--customAppCmd", fmt.Sprintf("%q", fmt.Sprintf("%s -R --nodaemonize --fpm-config %s", defaultFPMBinary, fpmConfFile.Name())),
			"--pid1LogFilePath", filepath.Join(l.Path, pid1Log),
			// Ideally, we should be able to use the path of the nginx layer and not hardcode it here.
			// This needs some investigation on how to pass values between build steps of buildpacks.
			"--mimeTypesPath", filepath.Join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
		}
		addArgs, err := addNginxConfCmdArgs(l.Path, nginxServerConfFile.Name(), overrides)
		if err != nil {
			return err
		}
		cmd = append(cmd, addArgs...)

		ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())
	}

	return nil
}

func getInstalledPhpVersion(ctx *gcp.Context) (string, error) {
	version, err := php.ExtractVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("determining runtime version: %w", err)
	}

	resolvedVersion, err := runtime.ResolveVersion(php.GetInstallableRuntime(ctx), version, runtime.OSForStack(ctx))
	if err != nil {
		return "", fmt.Errorf("resolving runtime version: %w", err)
	}

	return resolvedVersion, nil
}

func supportsDecorateWorkersOutput(ctx *gcp.Context) (bool, error) {
	v, err := getInstalledPhpVersion(ctx)
	if err != nil {
		return false, err
	}

	// Only latest php versions post 8.3 version are tested with RC candidate
	// and will support the below constraint (>= 7.3.0).
	if runtime.IsReleaseCandidate(v) {
		return true, nil
	}

	c, err := semver.NewConstraint(">= 7.3.0")
	if err != nil {
		return false, err
	}
	sv, err := semver.NewVersion(v)
	if err != nil {
		return false, fmt.Errorf("parsing semver: %w", err)
	}
	return c.Check(sv), nil
}

func writeFpmConfig(ctx *gcp.Context, path string, overrides webconfig.OverrideProperties) (*os.File, error) {
	// For php >= 7.3.0, the directive decorate_workers_output prevents php from prepending a warning
	// message to all logged entries.  Prior to 7.3.0, decorate_workers_output was not available, and
	// these warning messages are prepended to all logged entries.  Here we choose to set
	// decorate_workers_output if the runtime version is >= 7.3.0.
	addNoDecorateWorkers, err := supportsDecorateWorkersOutput(ctx)
	if err != nil {
		return nil, err
	}
	conf, err := fpmConfig(path, addNoDecorateWorkers, overrides)
	if err != nil {
		return nil, err
	}
	return nginx.WriteFpmConfigToPath(path, conf)
}

func fpmConfig(layer string, addNoDecorateWorkers bool, overrides webconfig.OverrideProperties) (nginx.FPMConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nginx.FPMConfig{}, fmt.Errorf("getting current user: %w", err)
	}

	fpm := nginx.FPMConfig{
		PidPath:              filepath.Join(layer, phpFpmPid),
		NumWorkers:           defaultFPMWorkers,
		ListenAddress:        filepath.Join(layer, appSocket),
		DynamicWorkers:       defaultDynamicWorkers,
		Username:             user.Username,
		AddNoDecorateWorkers: addNoDecorateWorkers,
	}

	if env.IsFlex() {
		fpm.ListenAddress = defaultFlexAddress
	}

	if overrides.PHPFPMOverride {
		fpm.ConfOverride = overrides.PHPFPMOverrideFileName
	}

	return fpm, nil
}

func addNginxConfCmdArgs(path, nginxServerConfFileName string, overrides webconfig.OverrideProperties) ([]string, error) {
	var args []string
	if env.IsFlex() {
		args = []string{"--customAppPort", "9000"}
	} else {
		args = []string{"--customAppSocket", filepath.Join(path, appSocket)}
	}

	if overrides.NginxConfOverride {
		return append(args, "--nginxConfigPath", overrides.NginxConfOverrideFileName), nil
	}

	args = append(args,
		"--nginxConfigPath", filepath.Join(path, nginxConf),
		"--serverConfigPath", nginxServerConfFileName,
	)

	if overrides.NginxHTTPInclude {
		args = append(args, "--httpIncludeConfigPath", overrides.NginxHTTPIncludeFileName)
	}

	return args, nil
}

func nginxConfig(layer string, overrides webconfig.OverrideProperties) nginx.Config {
	frontController := defaultFrontController
	if overrides.FrontController != "" {
		frontController = overrides.FrontController
	}

	root := defaultRoot
	if overrides.DocumentRoot != "" {
		root = filepath.Join(defaultRoot, overrides.DocumentRoot)
	}

	nginx := nginx.Config{
		Port:                  defaultNginxPort,
		FrontControllerScript: frontController,
		Root:                  root,
		AppListenAddress:      "unix:" + filepath.Join(layer, appSocket),
	}

	if env.IsFlex() {
		nginx.AppListenAddress = defaultFlexAddress
	}

	if overrides.NginxServerConfInclude {
		nginx.NginxConfInclude = overrides.NginxServerConfIncludeFileName
	}

	return nginx
}

func writeNginxServerConfig(path string, overrides webconfig.OverrideProperties) (*os.File, error) {
	conf := nginxConfig(path, overrides)
	return nginx.WriteNginxConfigToPath(path, conf)
}
