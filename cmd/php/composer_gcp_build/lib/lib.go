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

"""Implements php/composer_gcp_build buildpack."""

import json
import os
import subprocess

from gcpbuildpack import base


def detect(context):
    """Detects whether this buildpack should be applied."""
    if not context.file_exists('composer.json'):
        raise base.OptOutError('composer.json not found')
    
    try:
        with open(os.path.join(context.application_root, 'composer.json'), 'r') as f:
            composer = json.load(f)
            if not composer.get('scripts', {}).get('gcp-build'):
                raise base.OptOutError('gcp-build script not found in composer.json')
    except Exception as e:
        raise base.OptOutError(f'Error reading composer.json: {e}') from e
    
    return base.DetectResult(opt_in=True, message='Found composer.json with a gcp-build script')

def build(context):
    """Runs the build steps for this buildpack."""
    # Run Composer install
    try:
        subprocess.run(
            ['composer', 'install'],
            cwd=context.application_root,
            check=True,
            env={**os.environ, 'COMPOSER_CACHE_DIR': os.path.join(context.cache_dir, 'gcp-build-dependencies')}
        )
    except subprocess.CalledProcessError as e:
        raise base.BuildError(f'Composer install failed: {e}') from e
    
    # Run gcp-build script
    try:
        result = subprocess.run(
            ['composer', 'run-script', '--timeout=600', '--no-dev', 'gcp-build'],
            cwd=context.application_root,
            check=True,
            capture_output=True,
            text=True
        )
        print(f'gcp-build output:\n{result.stdout}')
    except subprocess.CalledProcessError as e:
        raise base.BuildError(f'Running gcp-build failed: {e}') from e
    
    # Cleanup vendor directory
    try:
        if os.path.exists(os.path.join(context.application_root, 'vendor')):
            shutil.rmtree(os.path.join(context.application_root, 'vendor'))
    except Exception as e:
        raise base.BuildError(f'Failed to cleanup vendor directory: {e}') from e
