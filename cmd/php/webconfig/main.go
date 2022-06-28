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

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
)

const (
	// pid1
	appSocket = "app.sock"
	pid1Log   = "pid1.log"

	// nginx
	defaultFrontController = "index.php"
	defaultNginxBinary     = "nginx"
	defaultNginxPort       = 8080
	defaultRoot            = "/workspace"
	nginxConf              = "nginx.conf"
	nginxLog               = "nginx.log"
	nginxServerConf        = "nginxserver.conf"

	// php-fpm
	defaultDynamicWorkers = false
	defaultFPMBinary      = "php-fpm"
	defaultFPMWorkers     = 2
	phpFpmConf            = "php-fpm.conf"
	phpFpmPid             = "php-fpm.pid"
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

	fpmConfFile, err := writeFpmConfig(ctx, l.Path)
	if err != nil {
		return err
	}
	defer fpmConfFile.Close()

	nginxConfFile, err := writeNginxConfig(ctx, l.Path)
	if err != nil {
		return err
	}
	defer nginxConfFile.Close()

	cmd := []string{
		"pid1",
		"--nginxBinaryPath", defaultNginxBinary,
		"--nginxConfigPath", filepath.Join(l.Path, nginxConf),
		"--serverConfigPath", nginxConfFile.Name(),
		"--nginxErrLogFilePath", filepath.Join(l.Path, nginxLog),
		"--customAppCmd", fmt.Sprintf("%q", fmt.Sprintf("%s -R --nodaemonize --fpm-config %s", defaultFPMBinary, fpmConfFile.Name())),
		"--pid1LogFilePath", filepath.Join(l.Path, pid1Log),
		// ideally, we should be able to use the path of the nginx layer and not hardcode it here.
		// this needs some investigation on how to pass values between build steps of buildpacks.
		"--mimeTypesPath", filepath.Join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
		"--customAppSocket", filepath.Join(l.Path, appSocket),
	}

	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}

func writeFpmConfig(ctx *gcp.Context, path string) (*os.File, error) {
	conf, err := fpmConfig(path)
	if err != nil {
		return nil, err
	}

	fpmConfFilePath := filepath.Join(path, phpFpmConf)
	fpmConfFile, err := os.Create(fpmConfFilePath)
	if err != nil {
		return nil, err
	}
	if err := nginx.PHPFpmTemplate.Execute(fpmConfFile, conf); err != nil {
		return nil, fmt.Errorf("writing php-fpm config file: %w", err)
	}
	return fpmConfFile, nil
}

func fpmConfig(l string) (nginx.FPMConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nginx.FPMConfig{}, fmt.Errorf("getting current user: %w", err)
	}

	fpm := nginx.FPMConfig{
		PidPath:        filepath.Join(l, phpFpmPid),
		NumWorkers:     defaultFPMWorkers,
		ListenAddress:  filepath.Join(l, appSocket),
		DynamicWorkers: defaultDynamicWorkers,
		Username:       user.Username,
	}

	return fpm, nil
}

func nginxConfig(l string) nginx.Config {
	nginx := nginx.Config{
		Port:                  defaultNginxPort,
		FrontControllerScript: defaultFrontController,
		Root:                  defaultRoot,
		AppListenAddress:      filepath.Join(l, appSocket),
	}

	return nginx
}

func writeNginxConfig(ctx *gcp.Context, path string) (*os.File, error) {
	nginxConfFilePath := filepath.Join(path, nginxServerConf)
	nginxConfFile, err := os.Create(nginxConfFilePath)
	if err != nil {
		return nil, err
	}

	conf := nginxConfig(path)
	if err := nginx.NginxTemplate.Execute(nginxConfFile, conf); err != nil {
		return nil, fmt.Errorf("writing nginx config file: %w", err)
	}
	return nginxConfFile, nil
}
