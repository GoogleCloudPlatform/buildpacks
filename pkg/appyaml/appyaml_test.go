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
from pathlib import Path
import unittest

import yaml

from pkg.appyaml import (
    appyaml_if_exists,
    entrypoint_if_exists,
    php_configuration,
)

class TestEntrypointIfExists(unittest.TestCase):
    def setUp(self):
        self.temp_dir = Path.cwd() / "temp_test"
        self.temp_dir.mkdir(exist_ok=True)
    
    def tearDown(self):
        for f in self.temp_dir.glob("*"):
            f.unlink()
        self.temp_dir.rmdir()

    def test_no_env_var(self):
        os.environ.pop('GAE_APPLICATION_YAML_PATH', None)
        result, err = entrypoint_if_exists(str(self.temp_dir))
        self.assertIsNone(result)
        self.assertIsNone(err)

    def test_valid_entrypoint(self):
        content = yaml.dump({'entrypoint': 'my entrypoint'})
        appyaml_path = self.temp_dir / 'app.yaml'
        appyaml_path.write_text(content, encoding='utf-8')
        os.environ['GAE_APPLICATION_YAML_PATH'] = str(apppyaml_path)
        result, err = entrypoint_if_exists(str(self.temp_dir))
        self.assertEqual(result, 'my entrypoint')
        self.assertIsNone(err)

    def test_missing_entrypoint(self):
        content = yaml.dump({'foo': 'bar'})
        appyaml_path = self.temp_dir / 'app.yaml'
        appyaml_path.write_text(content, encoding='utf-8')
        os.environ['GAE_APPLICATION_YAML_PATH'] = str(apppyaml_path)
        result, err = entrypoint_if_exists(str(self.temp_dir))
        self.assertIsNone(result)
        self.assertEqual(err, "Couldn't find entrypoint from app.yaml")

class TestPhpConfiguration(unittest.TestCase):
    def setUp(self):
        self.temp_dir = Path.cwd() / "temp_test"
        self.temp_dir.mkdir(exist_ok=True)
    
    def tearDown(self):
        for f in self.temp_dir.glob("*"):
            f.unlink()
        self.temp_dir.rmdir()

    def test_valid_runtime_config(self):
        content = yaml.dump({
            'runtime_config': {
                'document_root': 'web'
            }
        })
        appyaml_path = self.temp_dir / 'app.yaml'
        appyaml_path.write_text(content, encoding='utf-8')
        os.environ['GAE_APPLICATION_YAML_PATH'] = str(apppyaml_path)
        config, err = php_configuration(str(self.temp_dir))
        self.assertEqual(config.document_root, 'web')
        self.assertIsNone(err)

    def test_missing_runtime_config(self):
        appyaml_path = self.temp_dir / 'app.yaml'
        appyaml_path.write_text('', encoding='utf-8')
        os.environ['GAE_APPLICATION_YAML_PATH'] = str(apppyaml_path)
        config, err = php_configuration(str(self.temp_dir))
        self.assertEqual(config.document_root, '')
        self.assertIsNone(err)

if __name__ == '__main__':
    unittest.main()
