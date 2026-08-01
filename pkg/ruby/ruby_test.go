# Copyright 2022 Google LLC
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

import os
import tempfile

from typing import List, Tuple

class LockFile:
    def __init__(self, name: str, content: str):
        self.name = name
        self.content = content

def detect_ruby_version(ctx) -> str:
    """
    Detects the Ruby version from various sources.
    
    Args:
    ctx (Context): The context object containing application root and other metadata.

    Returns:
    str: The detected Ruby version.

    Raises:
    ValueError: If multiple versions are specified and conflict occurs.
    """

    # Get environment variable for runtime version
    env_version = os.environ.get('GOOGLE_RUNTIME_VERSION')

    # Create temporary directory to store lock files
    temp_root = ctx.application_root

    test_cases = [
        {
            'name': 'from environment',
            'runtime_env': env_version,
            'want': env_version,
        },
        {
            'name': 'from Gemfile.lock',
            'lock_files': [LockFile('Gemfile.lock', '''
RUBY VERSION
   ruby 2.5.7p206
''')],
            'want': '2.5.7',
        },
        {
            'name': 'from .ruby-version',
            'runtime_env': env_version,
            'lock_files': [LockFile('.ruby-version', '3.0.5')],
            'want': '3.0.5',
        },
        # Add more test cases as needed
    ]

    for tc in test_cases:
        if tc['runtime_env'] != "":
            os.environ['GOOGLE_RUNTIME_VERSION'] = tc['runtime_env']

        temp_root = tempfile.mkdtemp()
        lock_files_dir = os.path.join(temp_root, 'lock_files')
        os.mkdir(lock_files_dir)

        for lock_file in tc.get('lock_files', []):
            lock_file_path = os.path.join(lock_files_dir, lock_file.name)
            with open(lock_file_path, 'w') as f:
                f.write(lock_file.content)

        # Simulate context object (not used in this example)
        ctx = Context(application_root=temp_root)

        try:
            ruby_version = detect_ruby_version(ctx)
        except ValueError as e:
            if tc['error_content']:
                assert str(e) == tc['error_content']
            else:
                raise

        if ruby_version != tc['want']:
            assert False, f"Expected {tc['want']} but got {ruby_version}"

    return None

class Context:
    def __init__(self, application_root: str):
        self.application_root = application_root
