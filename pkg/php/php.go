import json
import os
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional

# Constants
composerJSON = "composer.json"
composerLock = "composer.lock"
Vendor = "vendor"

phpVersionKey = "php_version"
dependencyHashKey = "dependency_hash"

ComposerArgsEnv = "GOOGLE_COMPOSER_ARGS"
ComposerVersionEnv = "GOOGLE_COMPOSER_VERSION"
CustomNginxConfigEnv = "GOOGLE_CUSTOM_NGINX_CONFIG"
NginxServesStaticFilesEnv = "NGINX_SERVES_STATIC_FILES"

PHPIni = """
; Copyright 2022 Google Inc.
;
; Licensed under the Apache License, Version 2.0 (the "License");
; you may not use this file except in compliance with the License.
; You may obtain a copy of the License at
;
;     http://www.apache.org/licenses/LICENSE-2.0
;
; Unless required by applicable law or agreed to in writing, software
; distributed under the License is distributed on an "AS IS" BASIS,
; WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
; See the License for the specific language governing permissions and
; limitations under the License.

expose_php = Off
memory_limit = -1
max_execution_time = 0

;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
; Error handling and logging, based on php.ini-production. ;
;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;

error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
display_errors = Off
display_startup_errors = Off
log_errors = On
log_errors_max_len = 0
ignore_repeated_errors = Off
ignore_repeated_source = Off
html_errors = Off
zend.assertions = -1
;; Enable maximum file sizes up to Front-End limits.
upload_max_filesize = 32M
post_max_size = 32M
"""

@dataclass
class ComposerScriptsJSON:
    gcp_build: str

@dataclass
class ComposerJSON:
    require: Dict[str, str]
    scripts: ComposerScriptsJSON

def supports_app_engine_apis(ctx: dict) -> bool:
    runtime = os.getenv("RUNTIME")
    if runtime == "php55":
        return True
    # Assuming appengine is a module with the required function
    return appengine.apisEnabled(ctx)

def read_composer_json(dir_path: str = ".") -> Optional[ComposerJSON]:
    file_path = Path(dir_path) / composerJSON
    try:
        with open(file_path, 'r') as f:
            content = json.load(f)
            scripts = ComposerScriptsJSON(gcp_build=content.get('scripts', {}).get('gcp-build', ''))
            return ComposerJSON(
                require=content.get('require', {}),
                scripts=scripts
            )
    except FileNotFoundError:
        raise RuntimeError(f"Could not find {composerJSON}")
    except json.JSONDecodeError as e:
        raise ValueError(f"Failed to parse {composerJSON}: {e}")

def version(ctx: dict) -> str:
    result = subprocess.run(["php", "-r", "echo PHP_VERSION;"], capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError("Could not get PHP version")
    return result.stdout.strip()

def composer_install(ctx: dict, flags: List[str]) -> None:
    cmd = ["composer", "install"] + flags
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Composer install failed: {result.stderr}")

def composer_dump_autoload(ctx: dict, flags: List[str]) -> None:
    cmd = ["composer", "dump-autoload"] + flags
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Composer dump autoload failed: {result.stderr}")

def composer_install_layer(ctx: dict, cache_tag: str) -> Dict:
    # Implementation details would depend on the specific layer management
    pass

def extract_version(ctx: dict) -> Optional[str]:
    runtime_version = os.getenv("RUNTIME_VERSION")
    if runtime_version:
        print(f"Using runtime version from environment: {runtime_version}")
        return runtime_version
    
    composer_path = Path(ctx.get('application_root', '.')) / composerJSON
    if not composer_path.exists():
        return None

    try:
        cjs = read_composer_json(composer_path.parent)
        php_version = cjs.require.get(phpVersionKey)
        if php_version:
            print(f"Using PHP version from {composerJSON}: {php_version}")
            return php_version
        print("Composer.json exists but does not specify a PHP version")
    except Exception as e:
        print(f"Error reading composer.json: {e}")

    return None

def main():
    # Example usage
    ctx = {
        'application_root': '.'
    }
    try:
        cjs = read_composer_json()
        print(cjs)
        ver = extract_version(ctx)
        if ver:
            print(f"Detected PHP version: {ver}")
        else:
            print("Could not detect PHP version")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
