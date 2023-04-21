// Implements php/flex buildpack.
// The flex buildpack installs the config needed for PHP runtime.
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

	// php-fpm
	defaultDynamicWorkers = false
	defaultFPMBinary      = "php-fpm"
	defaultFPMWorkers     = 2
	phpFpmPid             = "php-fpm.pid"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if env.IsFlex() {
		return gcp.OptInEnvSet(env.XGoogleTargetPlatform), nil
	}

	return gcp.OptOutEnvNotSet(env.XGoogleTargetPlatform), nil
}

func buildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("flex", gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch)
	if err != nil {
		return err
	}
	yamlConfig, err := appyaml.PhpConfiguration(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	nginxConfFile, err := writeNginxConfig(l.Path, yamlConfig)
	if err != nil {
		return err
	}

	fpmConfFile, err := writeFpmConfig(l.Path, yamlConfig)
	if err != nil {
		return err
	}

	cmd := []string{
		"pid1",
		"--nginxBinaryPath", defaultNginxBinary,
		"--nginxConfigPath", filepath.Join(l.Path, nginxConf),
		"--serverConfigPath", nginxConfFile.Name(),
		"--nginxErrLogFilePath", filepath.Join(l.Path, nginxLog),
		"--customAppCmd", fmt.Sprintf("%q", fmt.Sprintf("%s -R --nodaemonize --fpm-config %s", defaultFPMBinary, fpmConfFile.Name())),
		"--pid1LogFilePath", filepath.Join(l.Path, pid1Log),
		// Ideally, we should be able to use the path of the nginx layer and not hardcode it here.
		// This needs some investigation on how to pass values between build steps of buildpacks.
		"--mimeTypesPath", filepath.Join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
		"--customAppSocket", filepath.Join(l.Path, appSocket),
	}

	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())

	return nil
}

func nginxConfig(l string, yamlConfig appyaml.RuntimeConfig) nginx.Config {
	nginx := nginx.Config{
		Port:                  defaultNginxPort,
		FrontControllerScript: defaultFrontController,
		Root:                  filepath.Join(defaultRoot, yamlConfig.DocumentRoot),
		AppListenAddress:      filepath.Join(l, appSocket),
	}

	return nginx
}

func writeNginxConfig(path string, yamlConfig appyaml.RuntimeConfig) (*os.File, error) {
	nginxConf := nginxConfig(path, yamlConfig)
	nginxConfFile, err := nginx.WriteNginxConfigToPath(path, nginxConf)
	if err != nil {
		return nil, err
	}
	return nginxConfFile, nil
}

func fpmConfig(l string) (nginx.FPMConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nginx.FPMConfig{}, fmt.Errorf("getting current user: %w", err)
	}

	fpm := nginx.FPMConfig{
		PidPath:              filepath.Join(l, phpFpmPid),
		NumWorkers:           defaultFPMWorkers,
		ListenAddress:        filepath.Join(l, appSocket),
		DynamicWorkers:       defaultDynamicWorkers,
		Username:             user.Username,
		AddNoDecorateWorkers: true,
	}

	return fpm, nil
}
func writeFpmConfig(path string, yamlConfig appyaml.RuntimeConfig) (*os.File, error) {
	conf, err := fpmConfig(path)
	if err != nil {
		return nil, err
	}
	return nginx.WriteFpmConfigToPath(path, conf)
}
