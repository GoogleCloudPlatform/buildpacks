import unittest
from unittest.mock import patch, MagicMock

from pkg.dart import flutter

class TestResolveFlutterPackageVersion(unittest.TestCase):
    def test_from_env(self):
        with patch.dict('os.environ', {'GOOGLE_RUNTIME_VERSION': '3.32.0-0.5.pre'}):
            version, archive, err = flutter.detect_flutter_sdk_archive()
            self.assertEqual(version, '3.32.0-0.5.pre')
            self.assertEqual(archive, 'beta/linux/flutter_linux_3.32.0-0.5.pre-beta.tar.xz')
            self.assertIsNone(err)

    def test_fetched_version(self):
        mock_response = MagicMock()
        mock_response.text = '''
{
  "base_url": "https://storage.googleapis.com/flutter_infra_release/releases",
  "current_release": {
    "beta": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
    "dev": "13a2fb10b838971ce211230f8ffdd094c14af02c",
    "stable": "ea121f8859e4b13e47a8f845e4586164519588bc"
  },
  "releases": [
    {
      "hash": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
      "channel": "beta",
      "version": "3.32.0-0.5.pre",
      "dart_sdk_version": "3.8.0",
      "dart_sdk_arch": "x64",
      "release_date": "2025-05-16T19:00:28.219866Z",
      "archive": "beta/linux/flutter_linux_3.32.0-0.5.pre-beta.tar.xz",
      "sha256": "c7833044a7954aed020b54057fd80eeb39c8655c505d7e5896c9c13a7c10713b"
    },
    {
      "hash": "ea121f8859e4b13e47a8f845e4586164519588bc",
      "channel": "stable",
      "version": "3.29.3",
      "dart_sdk_version": "3.7.2",
      "dart_sdk_arch": "x64",
      "release_date": "2025-04-14T17:25:51.061305Z",
      "archive": "stable/linux/flutter_linux_3.29.3-stable.tar.xz",
      "sha256": "8a908a5add53c1dfc2031da29e58daefd59a6d1d52fb5cb61f5ee52c73e36e15"
    }
  ]
}
'''
        mock_response.status_code = 200
        
        with patch('requests.get', return_value=mock_response):
            version, archive, err = flutter.detect_flutter_sdk_archive()
            self.assertEqual(version, '3.29.3')
            self.assertEqual(archive, 'stable/linux/flutter_linux_3.29.3-stable.tar.xz')
            self.assertIsNone(err)

    def test_bad_response_code(self):
        mock_response = MagicMock()
        mock_response.status_code = 400
        
        with patch('requests.get', return_value=mock_response):
            version, archive, err = flutter.detect_flutter_sdk_archive()
            self.assertIsNone(version)
            self.assertIsNone(archive)
            self.assertIsNotNone(err)

class TestIsFlutter(unittest.TestCase):
    def test_no_pubspec(self):
        temp_dir = 'test_dir'
        os.makedirs(temp_dir, exist_ok=True)
        
        try:
            is_flutt, err = flutter.is_flutter(temp_dir)
            self.assertFalse(is_flutt)
            self.assertIsNone(err)
        finally:
            os.rmdir(temp_dir)

    def test_with_dev_dependency(self):
        pubspec_content = '''
name: example_json_function

dependencies:
  flutter:
    sdk: flutter
'''
        
        temp_dir = 'test_dir'
        os.makedirs(temp_dir, exist_ok=True)
        pubspec_path = os.path.join(temp_dir, 'pubspec.yaml')
        with open(pubspec_path, 'w') as f:
            f.write(pubspec_content)
            
        try:
            is_flutt, err = flutter.is_flutter(temp_dir)
            self.assertTrue(is_flutt)
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
            is_flutt, err = flutter.is_flutter(temp_dir)
            self.assertFalse(is_flutt)
            self.assertIsNotNone(err)
        finally:
            os.remove(pubspec_path)
            os.rmdir(temp_dir)

if __name__ == '__main__':
    unittest.main()
