# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Implements utils/nginx buildpack.
The nginx buildpack installs the nginx web server, pid1 and serve binaries.
"""

import subprocess
import sys

def detect():
    """
    Detects if the nginx buildpack should be applied.
    Returns True if the buildpack should be used, False otherwise.
    """
    # Example detection logic (customize as needed)
    try:
        # Check for a file indicating the need for nginx
        with open('requirements.txt', 'r') as f:
            return 'nginx' in f.read()
    except FileNotFoundError:
        return False

def build():
    """
    Builds and installs the nginx environment.
    """
    # Install required packages
    try:
        subprocess.run([
            'apt-get', 'update'
        ], check=True)

        subprocess.run([
            'apt-get', 'install', '-y',
            'nginx',
            'libnginx-mod-http-auth-pam',
            'libnginx-mod-http-dav-ext',
            'libnginx-mod-http-echo',
            'libnginx-mod-http-form-inputs',
            'libnginx-mod-http-fp',
            'libnginx-mod-http-geoip',
            'libnginx-mod-http-image-filter',
            'libnginx-mod-http-perl',
            'libnginx-mod-http-subs-filter',
            'libnginx-mod-http-upstream-fair',
            'libnginx-mod-http-xslt-filter',
            'libnginx-mod-mail',
            'libnginx-mod-stream'
        ], check=True)

    except subprocess.CalledProcessError as e:
        print(f"Error installing nginx packages: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    if detect():
        build()
