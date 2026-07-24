# Complete refactored code here
import os
from typing import Dict

class PyProjectTest:
    def test_is_poetry_project(self):
        test_cases = [
            {
                "name": "poetry.lock_exists",
                "files": {"poetry.lock": "", "pyproject.toml": ""},
                "want": True,
                "want_msg": "found poetry.lock",
            },
            {
                "name": "pyproject.toml_with_tool.poetry_section",
                "files": {"pyproject.toml": '[tool.poetry]\nname = "my-test-project"'},
                "want": True,
                "want_msg": "found [tool.poetry] in pyproject.toml",
            },
            {
                "name": "pyproject.toml_without_tool.poetry_section",
                "files": {"pyproject.toml": '[tool.other]\nname = "my-test-project"'},
                "want": False,
                "want_msg": "neither poetry.lock nor [tool.poetry] found",
            },
            {
                "name": "no_relevant_files_exist",
                "files": {},
                "want": False,
                "want_msg": "pyproject.toml not found",
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            is_poetry, msg, err = self.is_poetry_project(ctx)

            if err != None:
                self.fail("IsPoetryProject() got an unexpected error: %v" % (err))

            if is_poetry != tc.want:
                self.assertEqual(is_poetry, tc.want)
                self.assertEqual(msg, tc.want_msg)

    def test_requested_poetry_version(self):
        test_cases = [
            {
                "name": "valid_requires-poetry_constraint",
                "files": {"pyproject.toml": '[tool.poetry]\nrequires-poetry = ">2.1.0"'},
                "want": ">2.1.0",
                "want_err": False,
            },
            {
                "name": "no_requires-poetry_constraint",
                "files": {"pyproject.toml": '[tool.poetry]'},
                "want": "",
                "want_err": False,
            },
            {
                "name": "malformed_pyproject.toml",
                "files": {"pyproject.toml": '[tool.poetry\nrequires-poetry = "<2.0.0"'},
                "want": "",
                "want_err": True,
            },
            {
                "name": "file_does_not_exist",
                "files": {},
                "want": "",
                "want_err": True,
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            version, err = self.requested_poetry_version(ctx)

            if (err != None) != tc.want_err:
                self.fail("RequestedPoetryVersion() error = %v, wantErr %v" % (err, tc.want_err))

            if err == None and version != tc.want:
                self.assertEqual(version, tc.want)

    def test_is_uv_pyproject(self):
        test_cases = [
            {
                "name": "uv_project_with_uv.lock",
                "files": {"pyproject.toml": '[project]\nname = "my-uv-project"', "uv.lock": ""},
                "want": True,
                "want_msg": "found pyproject.toml and uv.lock",
            },
            {
                "name": "uv_project_without_uv.lock",
                "files": {"pyproject.toml": '[project]\nname = "my-uv-project"'},
                "want": True,
                "want_msg": "found pyproject.toml and GOOGLE_PYTHON_PACKAGE_MANAGER is not set, using uv as default package manager",
            },
            {
                "name": "uv_project_without_uv.lock_with_uv_package_manager_env_var_set",
                "files": {"pyproject.toml": '[project]\nname = "my-uv-project"'},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "uv"},
                "want": True,
                "want_msg": "found pyproject.toml, using uv because GOOGLE_PYTHON_PACKAGE_MANAGER is set to 'uv'",
            },
            {
                "name": "uv_project_without_uv.lock_with_pip_package_manager_env_var_set",
                "files": {"pyproject.toml": '[project]\nname = "my-uv-project"'},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip"},
                "want": False,
                "want_msg": "found pyproject.toml, but GOOGLE_PYTHON_PACKAGE_MANAGER is not set to 'uv'",
            },
            {
                "name": "uv_project_with_uv.lock_with_pip_package_manager_env_var_set",
                "files": {"pyproject.toml": '[project]\nname = "my-uv-project"', "uv.lock": ""},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip"},
                "want": True,
                "want_msg": "found pyproject.toml and uv.lock",
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            if "env" in tc:
                for key, value in tc["env"].items():
                    os.environ[key] = value

            is_uv, msg, err = self.is_uv_pyproject(ctx)

            if err != None:
                self.fail("IsUVPyproject() got an unexpected error: %v" % (err))

            if is_uv != tc.want:
                self.assertEqual(is_uv, tc.want)
                self.assertEqual(msg, tc.want_msg)

    def test_requested_uv_version(self):
        test_cases = [
            {
                "name": "valid_required-version_constraint",
                "files": {"pyproject.toml": '[tool.uv]\nrequired-version = ">0.1.0"'},
                "want": ">0.1.0",
                "want_err": False,
            },
            {
                "name": "no_required-version_constraint",
                "files": {"pyproject.toml": '[tool.uv]'},
                "want": "",
                "want_err": False,
            },
            {
                "name": "malformed_pyproject.toml",
                "files": {"pyproject.toml": '[tool.uv\nrequired-version = "<1.0.0"'},
                "want": "",
                "want_err": True,
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            version, err = self.requested_uv_version(ctx)

            if (err != None) != tc.want_err:
                self.fail("RequestedUVVersion() error = %v, wantErr %v" % (err, tc.want_err))

            if err == None and version != tc.want:
                self.assertEqual(version, tc.want)

    def test_get_script_command(self):
        test_cases = [
            {
                "name": "poetry_single_script_found",
                "files": {"pyproject.toml": '[tool.poetry.scripts]\nstart_app = "my_app.main:run"'},
                "want": ["start_app"],
                "want_err": False,
            },
            {
                "name": "poetry_multiple_scripts_returns_start",
                "files": {"pyproject.toml": '[tool.poetry.scripts]\ndev = "my_app.dev:run"\nstart = "my_app.main:run"\nlint = "my_app.lint:run"'},
                "want": ["start"],
                "want_err": False,
            },
            {
                "name": "poetry_multiple_scripts_no_start_returns_nil",
                "files": {"pyproject.toml": '[tool.poetry.scripts]\ndev = "my_app.dev:run"\nlint = "my_app.lint:run"'},
                "want": None,
                "want_err": False,
            },
            {
                "name": "project_single_script_found",
                "files": {"pyproject.toml": '[project.scripts]\nstart_now = "my_app.main:run"'},
                "want": ["start_now"],
                "want_err": False,
            },
            {
                "name": "project_multiple_scripts_returns_start",
                "files": {"pyproject.toml": '[project.scripts]\ndev = "my_app.dev:run"\nstart = "my_app.main:run"'},
                "want": ["start"],
                "want_err": False,
            },
            {
                "name": "project_multiple_scripts_no_start_returns_nil",
                "files": {"pyproject.toml": '[project.scripts]\ndev = "my_app.dev:run"\nlint = "my_app.lint:run"'},
                "want": None,
                "want_err": False,
            },
            {
                "name": "poetry_single_script_takes_precedence_over_project_start",
                "files": {"pyproject.toml": '[tool.poetry.scripts]\nstart1 = "my_app.poetry:run"\n[project.scripts]\nstart2 = "my_app.project:run"\nstart = "my_app.project:start"'},
                "want": ["start1"],
                "want_err": False,
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            cmd, err = self.get_script_command(ctx)

            if (err != None) != tc.want_err:
                self.fail("GetScriptCommand() error = %v, wantErr %v" % (err, tc.want_err))

            if err == None and cmd != tc.want:
                self.assertEqual(cmd, tc.want)

    def test_is_pip_pyproject(self):
        test_cases = [
            {
                "name": "pip_pyproject_is_enabled_on_gcp",
                "files": {"pyproject.toml": "[project]"},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip", "X_GOOGLE_TARGET_PLATFORM": "gcp"},
                "want": True,
            },
            {
                "name": "disabled_when_requirements_txt_exists",
                "files": {"pyproject.toml": "[project]", "requirements.txt": "flask"},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip", "X_GOOGLE_RELEASE_TRACK": "BETA", "X_GOOGLE_TARGET_PLATFORM": "gcp"},
                "want": False,
            },
            {
                "name": "disabled_when_package_manager_is_uv",
                "files": {"pyproject.toml": "[project]"},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "uv", "X_GOOGLE_RELEASE_TRACK": "BETA", "X_GOOGLE_TARGET_PLATFORM": "gcp"},
                "want": False,
            },
            {
                "name": "disabled_when_no_package_manager",
                "files": {"pyproject.toml": "[project]"},
                "env": {"X_GOOGLE_RELEASE_TRACK": "BETA", "X_GOOGLE_TARGET_PLATFORM": "gcp"},
                "want": False,
            },
            {
                "name": "disabled_when_platform_is_not_gcp_or_gcf",
                "files": {"pyproject.toml": "[project]"},
                "env": {"GOOGLE_PYTHON_PACKAGE_MANAGER": "pip", "X_GOOGLE_RELEASE_TRACK": "BETA", "X_GOOGLE_TARGET_PLATFORM": "gae"},
                "want": False,
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            if "env" in tc:
                for key, value in tc["env"].items():
                    os.environ[key] = value

            got = self.is_pip_pyproject(ctx)

            if got != tc.want:
                self.assertEqual(got, tc.want)

    def test_is_pyproject_enabled(self):
        test_cases = [
            {
                "name": "enabled_with_only_pyproject_and_alpha_track",
                "files": {"pyproject.toml": "[project]"},
                "envs": {"X_GOOGLE_RELEASE_TRACK": "ALPHA"},
                "want": True,
            },
            {
                "name": "enabled_with_only_pyproject_and_beta_track",
                "files": {"pyproject.toml": "[project]"},
                "envs": {"X_GOOGLE_RELEASE_TRACK": "BETA"},
                "want": True,
            },
            {
                "name": "disabled_when_requirements_txt_exists",
                "files": {"requirements.txt": "flask", "pyproject.toml": "[project]"},
                "envs": {"X_GOOGLE_RELEASE_TRACK": "ALPHA"},
                "want": False,
            },
            {
                "name": "enabled_on_GA_track_for_python_313",
                "files": {"pyproject.toml": "[project]"},
                "envs": {"GOOGLE_RUNTIME_VERSION": "3.13.0"},
                "want": True,
            },
            {
                "name": "enabled_on_GA_track_for_python_314",
                "files": {"pyproject.toml": "[project]"},
                "envs": {"GOOGLE_RUNTIME_VERSION": "3.14.0"},
                "want": True,
            },
            {
                "name": "enabled_on_GA_track_for_universal_22",
                "files": {"pyproject.toml": "[project]"},
                "envs": {},
                "want": True,
            },
            {
                "name": "enabled_on_GA_track_for_python_312",
                "files": {"pyproject.toml": "[project]"},
                "envs": {"GOOGLE_RUNTIME_VERSION": "3.12.5"},
                "want": True,
            },
            {
                "name": "disabled_when_pyproject_toml_does_not_exist",
                "files": {"requirements.txt": "flask"},
                "envs": {"X_GOOGLE_RELEASE_TRACK": "ALPHA"},
                "want": False,
            },
        ]

        for tc in test_cases:
            app_dir = self.setup_test(tc.files)
            ctx = self.get_context(app_dir)

            if "envs" in tc:
                for key, value in tc["envs"].items():
                    os.environ[key] = value

            got_enabled = self.is_pyproject_enabled(ctx)

            if got_enabled != tc.want:
                self.assertEqual(got_enabled, tc.want)
