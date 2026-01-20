// Package faherror provides a standardized way to define and handle errors that occur during the
// Firebase App Hosting build process.
package faherror

import (
	"fmt"
)

// FahError is a wrapper around an error that provides additional metadata to help the user
// understand and resolve the error.
type FahError struct {
	Reason            string `json:"reason"`            // ex. 'Misconfigured Secret'
	Code              string `json:"code"`              // ex. 'fah/misconfigured-secret'
	UserFacingMessage string `json:"userFacingMessage"` // ex. 'Secret ... troubleshoot.'
	RawLog            string `json:"rawLog"`            // ex. 'calling out to secret manager...'
	IsUserAttributed  bool   `json:"isUserAttributed"`
}

func (e *FahError) Error() string {
	// We avoid using json.Marshal because it may return an error, which we can't handle in the
	// Error() method. Instead, we simply construct the JSON string directly.
	return fmt.Sprintf(`{"reason":"%v","code":"%v","userFacingMessage":"%v","rawLog":"%v","isUserAttributed":%t}`, e.Reason, e.Code, e.UserFacingMessage, e.RawLog, e.IsUserAttributed)
}

// ExitCode returns the exit code that the preparer/publisher should exit with.
func (e *FahError) ExitCode() int {
	if e.IsUserAttributed {
		return 100
	}
	return 1
}

// InternalErrorf covers internal Google-attributed errors.
func InternalErrorf(format string, args ...any) *FahError {
	err := fmt.Errorf(format, args...)
	return &FahError{
		Reason:            "Other Reason",
		Code:              "fah/other",
		UserFacingMessage: "Your build failed. Please check the raw log and build logs for more context about your build error.",
		RawLog:            err.Error(),
		IsUserAttributed:  false,
	}
}

// UserErrorf covers all other user-attributed errors that don't fit into the other known error types.
func UserErrorf(format string, args ...any) *FahError {
	err := fmt.Errorf(format, args...)
	return &FahError{
		Reason:            "Other Reason",
		Code:              "fah/other",
		UserFacingMessage: "Your build failed due to a misconfiguration. Please check the raw log and build logs for more context about your build error.",
		RawLog:            err.Error(),
		IsUserAttributed:  true,
	}
}

// MissingLockFileError creates a FahError with metadata about a missing lock file for the user's
// package manager (npm, yarn, or pnpm).
func MissingLockFileError(path string) *FahError {
	return &FahError{
		Reason:            "Missing Lock File",
		Code:              "fah/missing-lock-file",
		UserFacingMessage: fmt.Sprintf("Missing dependency lock file at path '%v'. Please run your package manager's install command and redeploy.", path),
		// We are generating the error source, so there's no raw log to include.
		RawLog:           "",
		IsUserAttributed: true,
	}
}

// MisconfiguredSecretError creates a FahError belonging to a class of errors that occur when a
// secret is either not found or permissions are not properly configured.
func MisconfiguredSecretError(secret string, rawLog error) *FahError {
	return &FahError{
		Reason: "Misconfigured Secret",
		Code:   "fah/misconfigured-secret",
		UserFacingMessage: fmt.Sprintf(
			"Error resolving secret version with name=%v. Please ensure the secret exists in your project and that your App Hosting backend has access to it. If the secret already exists in your project, please grant your App Hosting backend access to it with the CLI command 'firebase apphosting:secrets:grantaccess'. See https://firebase.google.com/docs/app-hosting/configure#secret-parameters for more information.",
			secret),
		RawLog:           rawLog.Error(),
		IsUserAttributed: true,
	}
}

// InvalidRootDirectoryError creates a FahError with metadata about a missing or invalid root
// directory that caused the build to fail.
func InvalidRootDirectoryError(rootDir string, rawLog error) *FahError {
	return &FahError{
		Reason:            "Invalid Root Directory",
		Code:              "fah/invalid-root-directory",
		UserFacingMessage: fmt.Sprintf("Invalid root directory specified. No buildable app found rooted at '%v'. Please go to your backend settings page and, in the Deployment tab, configure your root directory to point to the root of the target application.", rootDir),
		RawLog:            rawLog.Error(),
		IsUserAttributed:  true,
	}
}

// UnsupportedFrameworkVersionError creates a FahError with metadata about the unsupported framework
// version that caused the build to fail.
func UnsupportedFrameworkVersionError(framework string, version string) *FahError {
	return &FahError{
		Reason:            "Unsupported Framework Version",
		Code:              "fah/unsupported-framework-version",
		UserFacingMessage: fmt.Sprintf("Unsupported version for framework version %v@%v. Please see https://firebase.google.com/docs/app-hosting/about-app-hosting#frameworks for more information about which versions are supported by App Hosting.", framework, version),
		// We are generating the error source, so there's no raw log to include.
		RawLog:           "",
		IsUserAttributed: true,
	}
}

// FailedFrameworkBuildError creates a FahError belonging to the class of errors that occur when the
// framework build command fails.
func FailedFrameworkBuildError(buildCommand string, rawLog error) *FahError {
	return &FahError{
		Reason:            "Failed Framework Build",
		Code:              "fah/failed-framework-build",
		UserFacingMessage: fmt.Sprintf("Your application failed to run the framework build command '%v' successfully. Please check the raw log to address the error and confirm that your application builds locally before redeploying.", buildCommand),
		RawLog:            rawLog.Error(),
		IsUserAttributed:  true,
	}
}

// ImproperSecretFormatError creates a FahError with metadata about an improperly formatted secret
// in the user's apphosting.yaml that caused the build to fail.
func ImproperSecretFormatError(secret string) *FahError {
	return &FahError{
		Reason:            "Improper Secret Format",
		Code:              "fah/improper-secret-format",
		UserFacingMessage: fmt.Sprintf("Your secret '%s' is not formatted properly. Please see https://firebase.google.com/docs/app-hosting/configure#secret-parameters for guidance on how to format your secret.", secret),
		RawLog:            "",
		IsUserAttributed:  true,
	}
}

// InvalidAppHostingYamlError creates a FahError with metadata about an invalid apphosting.yaml
// file that caused the build to fail.
func InvalidAppHostingYamlError(filepath string, rawLog error) *FahError {
	return &FahError{
		Reason:            "Invalid apphosting.yaml",
		Code:              "fah/invalid-apphosting-yaml",
		UserFacingMessage: fmt.Sprintf("Your apphosting.yaml file at path '%v' is not formatted properly. Please see https://firebase.google.com/docs/app-hosting/configure#apphosting-yaml for guidance on how to format your apphosting.yaml file.", filepath),
		RawLog:            rawLog.Error(),
		IsUserAttributed:  true,
	}
}
