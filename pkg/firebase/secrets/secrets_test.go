# Complete refactored code here
"""
Tests for secrets package functionality.
"""

import contextlib
from unittest import TestCase, main
from unittest.mock import Mock

from google.cloud.secretmanager_v1.types import (
    AccessSecretVersionResponse,
    SecretVersion,
    SecretPayload,
)

import apphostingschema  # type: ignore
from faherror import ImproperSecretFormatError

import secrets


class TestNormalize(TestCase):
    """Tests for the normalize function"""

    def test_normalization(self):
        project_id = "test-project"
        env_vars = [
            apphostingschema.EnvironmentVariable(
                variable="API_URL",
                value="api.service.com",
                availability=["BUILD", "RUNTIME"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="SECRET_FORMAT_ONE",
                secret="secretID",
                availability=["BUILD"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="SECRET_FORMAT_TWO",
                secret="secretID@5",
                availability=["BUILD"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="SECRET_FORMAT_THREE",
                secret="projects/test-project/secrets/secretID",
                availability=["BUILD"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="SECRET_FORMAT_FOUR",
                secret="projects/test-project/secrets/secretID/versions/6",
                availability=["BUILD"]
            ),
        ]
        
        secrets.normalize(env_vars, project_id)
        
        self.assertEqual("api.service.com", env_vars[0].value)
        self.assertEqual(
            "projects/test-project/secrets/secretID/versions/latest",
            env_vars[1].secret
        )
        self.assertEqual(
            "projects/test-project/secrets/secretID/versions/5",
            env_vars[2].secret
        )
        self.assertEqual(
            "projects/test-project/secrets/secretID/versions/latest",
            env_vars[3].secret
        )
        self.assertEqual(
            "projects/test-project/secrets/secretID/versions/6",
            env_vars[4].secret
        )


class TestPinVersions(TestCase):
    """Tests for the pin_versions function"""

    def test_pin_versions(self):
        client = Mock()
        client.get_secret_version.return_value = "pinned-secret-name"

        env_vars = [
            apphostingschema.EnvironmentVariable(
                variable="PINNED_SECRET",
                secret="pinned-secret-name",
                availability=["BUILD"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="LATEST_SECRET",
                secret="latest-secret-name/versions/latest",
                availability=["BUILD"]
            ),
        ]

        secrets.pin_versions(env_vars, client)

        self.assertEqual("pinned-secret-name", env_vars[0].secret)
        self.assertEqual("pinned-secret-name", env_vars[1].secret)


class TestGenerateBuildDereferencedEnvMap(TestCase):
    """Tests for the generate_build_dereferenced_env_map function"""

    def test_generate_build_dereferenced_env_map(self):
        client = Mock()
        response = AccessSecretVersionResponse(
            payload=SecretPayload(
                data=b"secret-string",
                crc32c=crc32c.crc32(b"secret-string")
            )
        )
        client.access_secret_version.return_value = response

        env_vars = [
            apphostingschema.EnvironmentVariable(
                variable="API_URL",
                value="api.service.com",
                availability=["BUILD", "RUNTIME"]
            ),
            apphostingschema.EnvironmentVariable(
                variable="PINNED_SECRET",
                secret="pinned-secret-name",
                availability=["BUILD"]
            ),
        ]

        env_map = secrets.generate_build_dereferenced_env_map(env_vars, client)
        
        self.assertEqual("api.service.com", env_map["API_URL"])
        self.assertEqual("secret-string", env_map["PINNED_SECRET"])


if __name__ == "__main__":
    main()
