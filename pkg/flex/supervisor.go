package flex

import (
	"path"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

const (
	defaultSupervisorConf    = "supervisord.conf"
	defaultAddSupervisorConf = "additional-supervisord.conf"
)

// SupervisorConfig is the configuration for the template.
type SupervisorConfig struct {
	PHPFPMConfPath,
	NginxConfPath,
	SupervisorIncludeConfPath string
}

// SupervisorTemplate is a template that produces the supervisor configuration for Flex PHP applications
// to start the nginx process as part of backwards compatibility.
// Taken from https://github.com/GoogleCloudPlatform/php-docker/blob/master/php-base/supervisord.conf
var SupervisorTemplate = template.Must(template.New("supervisor").Parse(`
[supervisord]
nodaemon = true
logfile = /dev/null
logfile_maxbytes = 0

[program:php-fpm]
command = php-fpm -R --nodaemonize --fpm-config {{.PHPFPMConfPath}}
stdout_logfile = /dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile = /dev/stderr
stderr_logfile_maxbytes=0
autostart = true
autorestart = true
priority = 5

[program:nginx]
command = bash -c "sleep 1 && nginx -c {{.NginxConfPath}}"
stdout_logfile = /dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile = /dev/stderr
stderr_logfile_maxbytes=0
autostart = true
autorestart = true
priority = 10

[include]
files = {{.SupervisorIncludeConfPath}}
`))

// SupervisorFiles contains the necessary supervisor configuration files.
type SupervisorFiles struct {
	SupervisorConf          string
	AddSupervisorConf       string
	SupervisorConfExists    bool
	AddSupervisorConfExists bool
}

// SupervisorConfFiles returns whether it is necessary to install supervisor and the necessary files to fetch.
func SupervisorConfFiles(ctx *gcp.Context, runtimeConfig appyaml.RuntimeConfig, dir string) (SupervisorFiles, error) {
	supervisorConf, addSupervisorConf := defaultSupervisorConf, defaultAddSupervisorConf
	if runtimeConfig.SupervisordConfOverride != "" {
		supervisorConf = runtimeConfig.SupervisordConfOverride
	}

	if runtimeConfig.SupervisordConfAddition != "" {
		addSupervisorConf = runtimeConfig.SupervisordConfAddition
	}

	supervisorConfExists, err := ctx.FileExists(path.Join(dir, supervisorConf))
	if err != nil {
		return SupervisorFiles{}, err
	}

	addSupervisorConfExists, err := ctx.FileExists(path.Join(dir, addSupervisorConf))
	if err != nil {
		return SupervisorFiles{}, err
	}

	return SupervisorFiles{
		SupervisorConf:          supervisorConf,
		AddSupervisorConf:       addSupervisorConf,
		SupervisorConfExists:    supervisorConfExists,
		AddSupervisorConfExists: addSupervisorConfExists,
	}, nil
}

// NeedsSupervisorPackage returns whether to install supervisor.
func NeedsSupervisorPackage(ctx *gcp.Context) bool {
	runtimeConfig, err := appyaml.PhpConfiguration(ctx.ApplicationRoot())
	if err != nil {
		return false
	}
	files, err := SupervisorConfFiles(ctx, runtimeConfig, ctx.ApplicationRoot())
	if err != nil {
		return false
	}
	return files.SupervisorConfExists || files.AddSupervisorConfExists
}

// InstallSupervisor installs supervisor package to the layer.
func InstallSupervisor(ctx *gcp.Context, l *libcnb.Layer) error {
	l.LaunchEnvironment.Default("PYTHONUSERBASE", l.Path)
	if err := ctx.Setenv(
		"PYTHONUSERBASE", l.Path); err != nil {
		return err
	}

	cmd := []string{
		"python3", "-m", "pip", "install",
		"supervisor",
		"--upgrade",
		"--no-warn-script-location", // bin is added at run time by lifecycle.
		"--no-compile",
		"--disable-pip-version-check",
		"--no-cache-dir",
	}

	if _, err := ctx.Exec(cmd); err != nil {
		return err
	}

	return nil
}
