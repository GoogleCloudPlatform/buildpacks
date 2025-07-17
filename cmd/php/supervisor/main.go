// Implements php/supervisor buildpack.
// The supervisor buildpack installs the config needed for PHP runtime with supervisor.
package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/flex"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nginx"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/webconfig"
)

const (

	// nginx
	defaultFrontController      = "index.php"
	defaultNginxBinary          = "nginx"
	defaultNginxPort            = 8080
	defaultRoot                 = "/workspace"
	defaultNginxConfInclude     = "nginx-app.conf"
	defaultNginxConfHTTPInclude = "nginx-http.conf"
	defaultNginxConf            = "nginx.conf"
	nginxLog                    = "nginx.log"
	defaultAddress              = "127.0.0.1:9000"

	// php-fpm
	defaultPHPFPMConfOverride = "php-fpm.conf"
	defaultDynamicWorkers     = false
	defaultFPMBinary          = "php-fpm"
	defaultFPMWorkers         = 2
	phpFpmPid                 = "php-fpm.pid"

	defaultPHPIni = "php.ini"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if flex.NeedsSupervisorPackage(ctx) {
		return gcp.OptIn("supervisor package is required"), nil
	}

	return gcp.OptOut("supervisor package is not required"), nil
}

func buildFn(ctx *gcp.Context) error {
	l, err := ctx.Layer("supervisor", gcp.CacheLayer, gcp.LaunchLayerUnlessSkipRuntimeLaunch, gcp.BuildLayer)
	if err != nil {
		return err
	}
	l.LaunchEnvironment.Default("APP_DIR", defaultRoot)

	runtimeConfig, err := appyaml.PhpConfiguration(ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	if err = flex.InstallSupervisor(ctx, l); err != nil {
		return err
	}

	overrides := webconfig.OverriddenProperties(ctx, runtimeConfig)
	webconfig.SetEnvVariables(l, overrides)

	fpmConfFile, err := writeFpmConfig(l.Path, overrides)
	if err != nil {
		return err
	}

	supervisorFiles, err := flex.SupervisorConfFiles(ctx, runtimeConfig, ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	nginxPath := filepath.Join(l.Path, defaultNginxConf)
	// write the nginx configurations if they do not provide an override.
	if !overrides.NginxConfOverride {
		// nginx server section
		nginxServerConf, err := writeNginxServerConfig(l.Path, overrides)
		if err != nil {
			return err
		}
		// the nginx file to start the process.
		supervisorNginxConf := supervisorNginxConfig(nginxServerConf, overrides)

		if _, err = writeTemplateConfigToPath(nginxPath, flex.NginxConfTemplate, supervisorNginxConf); err != nil {
			return err
		}
	} else {
		nginxPath = overrides.NginxConfOverrideFileName
	}

	supervisorPath, err := supervisorLocation(supervisorFiles, nginxPath, fpmConfFile.Name(), l.Path)
	if err != nil {
		return err
	}
	cmd := []string{
		"supervisord", "-c", supervisorPath,
	}

	ctx.AddProcess(gcp.WebProcess, cmd, gcp.AsDefaultProcess())
	return nil
}

func supervisorLocation(supervisorFiles flex.SupervisorFiles, nginxPath, fpmConfFile, layer string) (string, error) {
	if supervisorFiles.SupervisorConfExists { // supervisord.conf overwritten
		return supervisorFiles.SupervisorConf, nil
	}

	// Generate the supervisord.conf otherwise.
	supervisorConf := supervisorConfig(fpmConfFile, nginxPath, supervisorFiles)
	supervisorFile, err := writeTemplateConfigToPath(filepath.Join(layer, "supervisord.conf"), flex.SupervisorTemplate, supervisorConf)
	if err != nil {
		return "", err
	}
	return supervisorFile.Name(), nil
}

func supervisorNginxConfig(nginxServerPath string, overrides webconfig.OverrideProperties) flex.NginxConfig {
	conf := flex.NginxConfig{
		MimeTypesPath:       filepath.Join("/layers/google.utils.nginx/nginx", "conf/mime.types"),
		NginxServerConfPath: nginxServerPath,
	}

	if overrides.NginxHTTPInclude {
		conf.NginxConfHTTPInclude = overrides.NginxHTTPIncludeFileName
	}
	return conf
}

func writeTemplateConfigToPath(path string, template *template.Template, config any) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := template.Execute(file, config); err != nil {
		return nil, fmt.Errorf("writing template config file: %w", err)
	}
	return file, nil
}

func writeNginxServerConfig(path string, overrides webconfig.OverrideProperties) (string, error) {
	nginxConf := nginxConfig(path, overrides)
	nginxConfFile, err := nginx.WriteNginxConfigToPath(path, nginxConf)
	if err != nil {
		return "", err
	}
	defer nginxConfFile.Close()
	return nginxConfFile.Name(), nil
}

func nginxConfig(layer string, overrides webconfig.OverrideProperties) nginx.Config {
	frontController := defaultFrontController
	if overrides.FrontController != "" {
		frontController = overrides.FrontController
	}

	nginx := nginx.Config{
		Port:                  defaultNginxPort,
		FrontControllerScript: frontController,
		Root:                  filepath.Join(defaultRoot, overrides.DocumentRoot),
		AppListenAddress:      defaultAddress,
	}

	if overrides.NginxServerConfInclude {
		nginx.NginxConfInclude = overrides.NginxServerConfIncludeFileName
	}

	return nginx
}

func supervisorConfig(fpmPath, nginxPath string, supervisorFiles flex.SupervisorFiles) flex.SupervisorConfig {
	supervisorConf := flex.SupervisorConfig{
		PHPFPMConfPath: fpmPath,
		NginxConfPath:  nginxPath,
	}

	if supervisorFiles.AddSupervisorConfExists {
		supervisorConf.SupervisorIncludeConfPath = filepath.Join(defaultRoot, supervisorFiles.AddSupervisorConf)
	}
	return supervisorConf
}

func fpmConfig(layer string, overrides webconfig.OverrideProperties) (nginx.FPMConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nginx.FPMConfig{}, fmt.Errorf("getting current user: %w", err)
	}

	fpm := nginx.FPMConfig{
		PidPath:              filepath.Join(layer, phpFpmPid),
		NumWorkers:           defaultFPMWorkers,
		ListenAddress:        defaultAddress,
		DynamicWorkers:       defaultDynamicWorkers,
		Username:             user.Username,
		AddNoDecorateWorkers: true,
		UseLogLimit:          true,
	}

	if overrides.PHPFPMOverride {
		fpm.ConfOverride = overrides.PHPFPMOverrideFileName
	}

	return fpm, nil
}

func writeFpmConfig(path string, overrides webconfig.OverrideProperties) (*os.File, error) {
	conf, err := fpmConfig(path, overrides)
	if err != nil {
		return nil, err
	}
	return nginx.WriteFpmConfigToPath(path, conf)
}
