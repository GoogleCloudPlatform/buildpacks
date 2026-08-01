import os
from pathlib import Path
from textwrap import dedent

class SupervisorConfig:
    def __init__(self, php_fpm_conf_path: str, nginx_conf_path: str, 
                 supervisor_include_conf_path: str):
        self.php_fpm_conf_path = php_fpm_conf_path
        self.nginx_conf_path = nginx_conf_path
        self.supervisor_include_conf_path = supervisor_include_conf_path

# SupervisorTemplate is a template that produces the supervisor configuration for Flex PHP applications
supervisor_template = dedent('''\
    [supervisord]
    nodaemon = true
    logfile = /dev/null
    logfile_maxbytes = 0

    [program:php-fpm]
    command = php-fpm -R --nodaemonize --fpm-config {php_fpm_conf_path}
    stdout_logfile = /dev/stdout
    stdout_logfile_maxbytes=0
    stderr_logfile = /dev/stderr
    stderr_logfile_maxbytes=0
    autostart = true
    autorestart = true
    priority = 5

    [program:nginx]
    command = bash -c "sleep 1 && nginx -c {nginx_conf_path}"
    stdout_logfile = /dev/stdout
    stdout_logfile_maxbytes=0
    stderr_logfile = /dev/stderr
    stderr_logfile_maxbytes=0
    autostart = true
    autorestart = true
    priority = 10

    [include]
    files = {supervisor_include_conf_path}
''').strip()

class SupervisorFiles:
    def __init__(self, supervisor_conf: str, add_supervisor_conf: str,
                 supervisor_conf_exists: bool, add_supervisor_conf_exists: bool):
        self.supervisor_conf = supervisor_conf
        self.add_supervisor_conf = add_supervisor_conf
        self.supervisor_conf_exists = supervisor_conf_exists
        self.add_supervisor_conf_exists = add_supervisor_conf_exists

def supervisor_conf_files(ctx: dict, runtime_config: dict, dir_path: str) -> SupervisorFiles:
    default_supervisor_conf = "supervisord.conf"
    default_add_supervisor_conf = "additional-supervisord.conf"

    supervisor_conf = runtime_config.get("SupervisordConfOverride", default_supervisor_conf)
    add_supervisor_conf = runtime_config.get("SupervisordConfAddition", default_add_supervisor_conf)

    supervisor_path = os.path.join(dir_path, supervisor_conf)
    supervisor_exists = Path(supervisor_path).exists()

    add_supervisor_path = os.path.join(dir_path, add_supervisor_conf)
    add_supervisor_exists = Path(add_supervisor_path).exists()

    return SupervisorFiles(
        supervisor_conf=supervisor_conf,
        add_supervisor_conf=add_supervisor_conf,
        supervisor_conf_exists=supervisor_exists,
        add_supervisor_conf_exists=add_supervisor_exists
    )

def needs_supervisor_package(ctx: dict) -> bool:
    runtime_config = ctx.get("appyaml", {}).get("RuntimeConfig", {})
    files, _ = supervisor_conf_files(ctx, runtime_config, ctx["application_root"])
    return files.supervisor_conf_exists or files.add_supervisor_conf_exists

def install_supervisor(ctx: dict, layer_path: str) -> None:
    env_vars = {
        "PYTHONUSERBASE": layer_path
    }
    
    cmd = [
        "python3", "-m", "pip", "install",
        "supervisor",
        "--upgrade",
        "--no-warn-script-location",
        "--no-compile",
        "--disable-pip-version-check",
        "--no-cache-dir"
    ]
    
    # In actual implementation, this would execute the command
    # subprocess.run(cmd, check=True)
