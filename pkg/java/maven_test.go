import json
from unittest.mock import patch
from unittest.mock import Mock
import requests


class TestGetLatestGradleVersion:
    @patch('requests.get')
    def test_latest(self, mock_get):
        mock_response = {
            'version': '7.4.2',
            'buildTime': '20220331152529+0000',
            'current': True,
            'snapshot': False,
            'nightly': False,
            'releaseNightly': False,
            'activeRc': False,
            'rcFor': '',
            'milestoneFor': '',
            'broken': False,
            'downloadUrl': 'https://services.gradle.org/distributions/gradle-7.4.2-bin.zip',
            'checksumUrl': 'https://services.gradle.org/distributions/gradle-7.4.2-bin.zip.sha256',
            'wrapperChecksumUrl': 'https://services.gradle.org/distributions/gradle-7.4.2-wrapper.jar.sha256'
        }
        mock_get.return_value.json.return_value = mock_response
        mock_get.return_value.status_code = 200

        gradle_version_url = 'https://example.com/gradle-version'
        with patch.object(requests, 'get') as mock_get:
            mock_get.return_value.__enter__.return_value.json.return_value = mock_response
            mock_get.return_value.__enter__.return_value.status_code = 200
            mock_get.return_value.__enter__.return_value.url = gradle_version_url

            got, err = get_latest_gradle_version()
            assert err is None
            assert got == '7.4.2'

    @patch('requests.get')
    def test_unavailable(self, mock_get):
        mock_response = {'message': 'not found'}
        mock_get.return_value.json.return_value = mock_response
        mock_get.return_value.status_code = 404

        gradle_version_url = 'https://example.com/gradle-version'
        with patch.object(requests, 'get') as mock_get:
            mock_get.return_value.__enter__.return_value.json.return_value = mock_response
            mock_get.return_value.__enter__.return_value.status_code = 404
            mock_get.return_value.__enter__.return_value.url = gradle_version_url

            got, err = get_latest_gradle_version()
            assert err is not None
            assert got is None


def stub_gradle_version_service(t):
    testserver.new(
        t,
        testserver.with_status(200),
        testserver.with_json(json.dumps({'version': '7.4.2'})),
        testserver.with_mock_url(gradle_version_url)
    )


class TestGetLatestGradleVersion:
    def test_latest(self):
        stub_gradle_version_service(None)

        got, err = get_latest_gradle_version()
        assert err is None
        assert got == '7.4.2'


def get_latest_gradle_version():
    # this function should be refactored to use the requests library
    pass


class TestGradleVersionService:
    def test_get_version(self):
        version = get_latest_gradle_version()
        assert version == '7.4.2'
