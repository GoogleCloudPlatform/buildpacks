import json
from typing import Any


class FahError(Exception):
    """
    A standardized way to define and handle errors during Firebase App Hosting builds.
    """

    def __init__(
        self,
        reason: str,
        code: str,
        user_facing_message: str,
        raw_log: str = "",
        is_user_attributed: bool = False
    ):
        super().__init__()
        self.reason = reason
        self.code = code
        self.user_facing_message = user_facing_message
        self.raw_log = raw_log
        self.is_user_attributed = is_user_attributed

    def __str__(self) -> str:
        error_dict = {
            "reason": self.reason,
            "code": self.code,
            "userFacingMessage": self.user_facing_message,
            "rawLog": self.raw_log,
            "isUserAttributed": self.is_user_attributed
        }
        return json.dumps(error_dict)

    def exit_code(self) -> int:
        """
        Returns the appropriate exit code based on whether the error is user-attributed.
        """
        return 100 if self.is_user_attributed else 1


def internal_errorf(format_str: str, *args: Any) -> FahError:
    message = format_str % args
    return FahError(
        reason="Other Reason",
        code="fah/other",
        user_facing_message=(
            "Your build failed. Please check the raw log and build logs for more "
            "context about your build error."
        ),
        raw_log=message,
        is_user_attributed=False
    )


def user_errorf(format_str: str, *args: Any) -> FahError:
    message = format_str % args
    return FahError(
        reason="Other Reason",
        code="fah/other",
        user_facing_message=(
            "Your build failed due to a misconfiguration. Please check the raw log and "
            "build logs for more context about your build error."
        ),
        raw_log=message,
        is_user_attributed=True
    )


def missing_lock_file_error(path: str) -> FahError:
    return FahError(
        reason="Missing Lock File",
        code="fah/missing-lock-file",
        user_facing_message=(
            f"Missing dependency lock file at path '{path}'. Please run your package "
            "manager's install command and redeploy."
        ),
        raw_log="",
        is_user_attributed=True
    )


def misconfigured_secret_error(secret: str, raw_log: str) -> FahError:
    return FahError(
        reason="Misconfigured Secret",
        code="fah/misconfigured-secret",
        user_facing_message=(
            f"Error resolving secret version with name={secret}. Please ensure the "
            "secret exists in your project and that your App Hosting backend has access "
            "to it. If the secret already exists in your project, please grant your App "
            "Hosting backend access to it with the CLI command 'firebase apphosting:"
            "secrets:grantaccess'. See https://firebase.google.com/docs/app-hosting/"
            "configure#secret-parameters for more information."
        ),
        raw_log=raw_log,
        is_user_attributed=True
    )


def invalid_root_directory_error(root_dir: str, raw_log: str) -> FahError:
    return FahError(
        reason="Invalid Root Directory",
        code="fah/invalid-root-directory",
        user_facing_message=(
            f"Invalid root directory specified. No buildable app found rooted at '{root_dir}'. "
            "Please go to your backend settings page and, in the Deployment tab, configure your "
            "root directory to point to the root of the target application."
        ),
        raw_log=raw_log,
        is_user_attributed=True
    )


def unsupported_framework_version_error(framework: str, version: str) -> FahError:
    return FahError(
        reason="Unsupported Framework Version",
        code="fah/unsupported-framework-version",
        user_facing_message=(
            f"Unsupported version for framework {framework}@{version}. Please see "
            "https://firebase.google.com/docs/app-hosting/about-app-hosting#frameworks for more "
            "information about which versions are supported by App Hosting."
        ),
        raw_log="",
        is_user_attributed=True
    )


def failed_framework_build_error(build_command: str, raw_log: str) -> FahError:
    return FahError(
        reason="Failed Framework Build",
        code="fah/failed-framework-build",
        user_facing_message=(
            f"Your application failed to run the framework build command '{build_command}' "
            "successfully. Please check the raw log to address the error and confirm that your "
            "application builds locally before redeploying."
        ),
        raw_log=raw_log,
        is_user_attributed=True
    )


def improper_secret_format_error(secret: str) -> FahError:
    return FahError(
        reason="Improper Secret Format",
        code="fah/improper-secret-format",
        user_facing_message=(
            f"Your secret '{secret}' is not formatted properly. Please see "
            "https://firebase.google.com/docs/app-hosting/configure#secret-parameters for guidance "
            "on how to format your secret."
        ),
        raw_log="",
        is_user_attributed=True
    )


def invalid_apphosting_yaml_error(filepath: str, raw_log: str) -> FahError:
    return FahError(
        reason="Invalid apphosting.yaml",
        code="fah/invalid-apphosting-yaml",
        user_facing_message=(
            f"Your apphosting.yaml file at path '{filepath}' is not formatted properly. Please see "
            "https://firebase.google.com/docs/app-hosting/configure#apphosting-yaml for guidance on how "
            "to format your apphosting.yaml file."
        ),
        raw_log=raw_log,
        is_user_attributed=True
    )
