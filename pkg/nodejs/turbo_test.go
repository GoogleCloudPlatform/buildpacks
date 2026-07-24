import unittest

class TestInstallPNPM(unittest.TestCase):
    def test_install_pnpm(self):
        # Assume mock tarball and npm registry responses are set up
        mock_tarball = b"pnpm!"
        
        with self.subTest("no_version_constraint"):
            pnpm_response = {
                "name": "pnpm",
                "dist-tags": {"latest": "11.0.0"},
                "versions": {"11.0.0": {"name": "pnpm", "version": "11.0.0"}},
                "modified": "2026-05-21T21:10:55.626Z"
            }
            
            install_pnpm(mock_tarball, pnpm_response)
        
        with self.subTest("valid_version_constraint"):
            pnpm_response = {
                "name": "pnpm",
                "dist-tags": {"latest": "8.4.0"},
                "versions": {"8.4.0": {"name": "npm", "version": "8.4.0"}},
                "modified": "2022-01-27T21:10:55.626Z"
            }
            
            install_pnpm(mock_tarball, pnpm_response)
        
        with self.subTest("invalid_version"):
            pnpm_response = {
                "name": "pnpm",
                "dist-tags": {"latest": "8.4.0"},
                "versions": {"8.4.0": {"name": "npm", "version": "8.4.0"}},
                "modified": "2022-01-27T21:10:55.626Z"
            }
            
            install_pnm_invalid_version(mock_tarball, pnpm_response)

class TestDetectPNPMVersion(unittest.TestCase):
    def test_detect_pnpm_version(self):
        # Assume mock npm registry response is set up
        pnpm_response = {
            "name": "pnpm",
            "dist-tags": {"latest": "9.2.0"},
            "versions": {"9.2.0": {"name": "npm", "version": "9.2.0"}},
            "modified": "2022-01-27T21:10:55.626Z"
        }
        
        version = detect_pnpm_version(pnpm_response, "ubuntu1804")
        
        self.assertEqual(version, "10.12.4")

class TestInstallPNPMV11(unittest.TestCase):
    def test_install_pnpmv11(self):
        # Assume mock tarball and npm registry responses are set up
        mock_tarball = b"pnpm!"
        
        pnpm_response = {
            "name": "pnpm",
            "dist-tags": {"latest": "11.0.0"},
            "versions": {"11.0.0": {"name": "pnpm", "version": "11.0.0"}}
        }
        
        install_pnpm(mock_tarball, pnpm_response)
