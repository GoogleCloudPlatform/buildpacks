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

from unittest import TestCase
import os
import sys
import tempfile

from googlecloudsdk.core import exceptions
from googlecloudsdk.core import properties
from googlecloudsdk.core.util import platforms
from functions_framework import base
from tests import buildpacktest

class TestDetect(TestCase):
  @classmethod
  def setUpClass(cls):
    cls.module_path = os.path.dirname(__file__)

  def testDetect(self):
    self.assertEqual(base.Detect().target, 'HelloWorld')

  def testNoTarget(self):
    self.assertEqual(base.Detect().target, None)

class TestBuild(TestCase):
  @classmethod
  def setUpClass(cls):
    cls.module_path = os.path.dirname(__file__)
    cls.app_name = 'myfunc'

  def testGoModFunctionWithFramework(self):
    envs = {'GOOGLE_FUNCTION_TARGET': 'Func'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)
    self.assertIn('go mod tidy', result.commands)

  def testGoModFunctionWithFrameworkWithoutInjection(self):
    envs = {'GOOGLE_SKIP_FRAMEWORK_INJECTION': 'True'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 1)

  def testGoModFunctionWithoutFramework(self):
    envs = {'GOOGLE_FUNCTION_TARGET': 'Func'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)
    self.assertIn('go mod edit -require %s' % functionsFrameworkModule, result.commands)
    self.assertIn('go mod tidy', result.commands)

  def testGoModFunctionWithoutFrameworkWithoutInjection(self):
    envs = {'GOOGLE_SKIP_FRAMEWORK_INJECTION': 'True'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 1)

  def testVendoredFunction(self):
    envs = {'GOOGLE_FUNCTION_TARGET': 'Func'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)
    self.assertIn('go mod vendor', result.commands)

  def testVendoredFunctionWithoutInjection(self):
    envs = {'GOOGLE_SKIP_FRAMEWORK_INJECTION': 'True'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 1)

  def testWithFrameworkVendoredForGo122AndBelow(self):
    envs = {'GOOGLE_RUNTIME_VERSION': '1.22.11'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
        self._mockProcess('go list -m -f {{.Version}}.*',
                           output='v1.0.0'),
        self._mockProcess('go version', output='go version go1.22 linux/amd64'),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)

  def testWithFrameworkVendoredForGo123AndAbove(self):
    envs = {'GOOGLE_RUNTIME_VERSION': '1.23.5'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
        self._mockProcess('go list -m -f {{.Version}}.*',
                           output='v1.0.0'),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)

  def testWithFrameworkVendoredForGo123AndAboveWithoutVersionSpecified(self):
    envs = {'GOOGLE_RUNTIME_VERSION': '1.23.5'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
        self._mockProcess('go list -m -f {{.Version}}.*',
                           output='v1.0.0'),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 0)

  def testWithFrameworkVendoredWithoutInjection(self):
    envs = {'GOOGLE_SKIP_FRAMEWORK_INJECTION': 'True'}
    mocks = [
        self._mockProcess(
            'get_package', output='{"name": "%s"}' % self.app_name),
        self._mockProcess('go list -m -f {{.Version}}.*',
                           output='v1.0.0'),
    ]
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 1)

  def testWithoutFrameworkVendored(self):
    envs = {'GOOGLE_FUNCTION_TARGET': 'Func'}
    mocks = []
    result, err = buildpacktest.RunBuild(self, base.Build(), {
        'app': self.app_name,
        'envs': envs,
        'execMocks': mocks,
    })
    self.assertEqual(result.exit_code, 1)

  def _mockProcess(self, command, output):
    return tempfile.NamedTemporaryFile(
        prefix='mock_',
        suffix='.txt').close(), sys.stdin.write(output.encode('utf-8'))

if __name__ == '__main__':
  unittest.main()
