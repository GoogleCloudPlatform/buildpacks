import unittest
from unittest.mock import patch, MagicMock

from pkg.dart import dart

class TestResolvePackageVersion(unittest.TestCase):
    def test_from_env(self):
        with patch.dict('os.environ', {'GOOGLE_RUNTIME_VERSION': '2.14.0'}):
            version, err = dart.detect_sdk_version()
            self.assertEqual(version, '2.14.0')
            self.assertIsNone(err)

    def test_fetched_version(self):
        mock_response = MagicMock()
        mock_response.text = '{"date": "2022-02-08", "version": "2.16.1", "revision": "0180af250ff518cc0fa494a4eb484ce11ec1e62c"}'
        mock_response.status_code = 200
        
        with patch('requests.get', return_value=mock_response):
            version, err = dart.detect_sdk_version()
            self.assertEqual(version, '2.16.1')
            self.assertIsNone(err)

    def test_bad_response_code(self):
        mock_response = MagicMock()
        mock_response.status_code = 400
        
        with patch('requests.get', return_value=mock_response):
            version, err = dart.detect_sdk_version()
            self.assertIsNone(version)
            self.assertIsNotNone(err)

class TestHasBuildRunner(unittest.TestCase):
    def test_no_pubspec(self):
        temp_dir = 'test_dir'
        os.makedirs(temp_dir, exist_ok=True)
        
        try:
            has_runner, err = dart.has_build_runner(temp_dir)
            self.assertFalse(has_runner)
            self.assertIsNone(err)
        finally:
            os.rmdir(temp_dir)

    def test_with_dev_dependency(self):
        pubspec_content = '''
name: example_json_function

dependencies:
  functions_framework: ^0.4.0

dev_dependencies:
  build_runner: ^2.0.0
'''
        
        temp_dir = 'test_dir'
        os.makedirs(temp_dir, exist_ok=True)
        pubspec_path = os.path.join(temp_dir, 'pubspec.yaml')
        with open(pubspec_path, 'w') as f:
            f.write(pubspec_content)
            
        try:
            has_runner, err = dart.has_build_runner(temp_dir)
            self.assertTrue(has_runner)
            self.assertIsNone(err)
        finally:
            os.remove(pubspec_path)
            os.rmdir(temp_dir)

    def test_invalid_yaml(self):
        pubspec_content = '\t'
        
        temp_dir = 'test_dir'
        os.makedirs(temp_dir, exist_ok=True)
        pubspec_path = os.path.join(temp_dir, 'pubspec.yaml')
        with open(pubspec_path, 'w') as f:
            f.write(pubspec_content)
            
        try:
            has_runner, err = dart.has_build_runner(temp_dir)
            self.assertFalse(has_runner)
            self.assertIsNotNone(err)
        finally:
            os.remove(pubspec_path)
            os.rmdir(temp_dir)

if __name__ == '__main__':
    unittest.main()
