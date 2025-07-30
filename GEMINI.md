# Gemini Code Understanding

This document outlines the structure of the project to help understand the codebase.

## Directory Structure

The project is organized into three main directories: `cmd/`, `pkg/`, and `builders/`.

### `cmd/`

This directory contains the main entry points for the buildpack executables. Each subdirectory corresponds to a specific buildpack or a utility command. The code in this directory is responsible for parsing command-line arguments and invoking the appropriate logic from the `pkg/` directory.

Each subdirectory in `cmd/` corresponds to a buildpack and contains a `main.go` file that serves as the entry point for that buildpack. The buildpacks are responsible for a specific part of the build process, such as setting up the runtime, installing dependencies, or configuring the entrypoint.

- **Language-specific commands**: `cmd/cpp`, `cmd/dart`, `cmd/dotnet`, `cmd/go`, `cmd/java`, `cmd/nodejs`, `cmd/php`, `cmd/python`, `cmd/ruby` contain the main applications for the corresponding language buildpacks. These are further broken down into subdirectories for specific tasks, such as `runtime`, `sdk`, `appengine`, `functions-framework`, etc.
- **Utility commands**: `cmd/config`, `cmd/utils` provide helper utilities. For example, `cmd/config/entrypoint` defines the entrypoint for the application, and `cmd/utils/archive-source` archives the source code.
- **Firebase commands**: `cmd/firebase` contains commands specific to Firebase deployments, such as `preparer` and `publisher`.

### `pkg/`

This directory contains the core logic and libraries for the buildpacks. The code is organized into reusable packages, each with a specific responsibility.

- **`gcpbuildpack`**: The core framework for creating buildpacks. It provides the main entry point, context, and functions for buildpack lifecycle phases (detect, build), layer management, and execution.
- **Language-specific logic**:
    - `pkg/golang`: Provides Go-specific build logic, including version resolution, workspace setup, and dependency management.
    - `pkg/java`: Contains Java build logic, with support for Maven and Gradle. It handles JAR discovery, manifest parsing, and dependency caching.
    - `pkg/nodejs`: Implements Node.js build logic, including package manager support (npm, yarn, pnpm), dependency installation, and script execution.
    - `pkg/php`: Provides PHP build logic, with support for Composer. It handles dependency installation and configuration.
    - `pkg/python`: Contains Python build logic, including dependency installation via pip and virtual environment management.
    - `pkg/ruby`: Implements Ruby build logic, with support for Bundler. It handles Gemfile parsing and dependency installation.
    - `pkg/dotnet`: Provides .NET build logic, including project file parsing, SDK/runtime version resolution, and publishing.
    - `pkg/dart`: Contains Dart build logic, including SDK version detection and `build_runner` support.
- **Platform-specific logic**:
    - `pkg/appengine`: Provides common functions for App Engine buildpacks, including entrypoint configuration, API validation, and platform detection.
    - `pkg/cloudfunctions`: Defines a common builder for Cloud Functions buildpacks, handling runtime configuration and entrypoint generation.
    - `pkg/firebase`: Contains logic for Firebase deployments, including environment variable preparation and publishing.
    - `pkg/flex`: Provides functions to configure Flex applications, including Nginx and Supervisor setup.
- **Common libraries**:
    - `pkg/appstart`: Creates the `app_start.json` config file for defining the application's entrypoint.
    - `pkg/appyaml`: Handles `app.yaml` configuration files for App Engine.
    - `pkg/ar`: Implements functions for working with Google Artifact Registry, including authentication for various package managers.
    - `pkg/buildererror`: Defines a structured error format for buildpacks.
    - `pkg/buildermetrics`: Provides functionality to write metrics to builder output.
    - `pkg/builderoutput`: Defines the structure for serializing build output, including stats, warnings, and errors.
    - `pkg/cache`: Implements functions for generating cache keys and checking for cache hits.
    - `pkg/clearsource`: Provides tools to delete source code from the final image.
    - `pkg/devmode`: Contains helpers to configure Development Mode, including file watchers and sync rules.
    - `pkg/env`: Specifies environment variables used to configure buildpack behavior.
    - `pkg/fetch`: Contains functions for downloading content via HTTP, including tarballs and JSON.
    - `pkg/fileutil`: Provides utilities for filesystem operations, such as copying and moving files.
    - `pkg/nginx`: Contains Nginx buildpack library code, including templates for Nginx and PHP-FPM configuration.
    - `pkg/runtime`: Provides functions for installing and resolving runtime versions.
    - `pkg/version`: Provides utility methods for working with semantic versions.
    - `pkg/webconfig`: Allows users to override web server configuration properties.

### `builders/`

This directory contains the configuration for the builders. Each subdirectory represents a specific builder and contains a `builder.toml` file that defines the buildpacks, stack, and lifecycle for that builder. The `builder.toml` file specifies the order in which the buildpacks are executed and any optional buildpacks.

- **Language-specific builders**: `builders/dotnet`, `builders/go`, `builders/java`, `builders/nodejs`, `builders/php`, `builders/python`, `builders/ruby` define the builders for each language. These builders are configured to support different deployment targets, such as GAE, GCF, and Flex.
- **Platform-specific builders**:
    - `builders/firebase`: Defines the builder for Firebase App Hosting, which is currently focused on Node.js applications.
    - `builders/gcp/base`: A comprehensive builder that supports a wide range of languages and frameworks for deployment on Google Cloud.
- **Templates**: Some builders, like `java` and `python`, use `builder.toml.template` files. These templates are used to generate the final `builder.toml` with specific stack information during the build process.

## Development Tasks

Given a prompt with a URL link which has language specific features and a request to update the code-base specific to that language version, your task is to update this code base to make sure that the Builders continue to work fine. Update the tests as you see fit but do not worry about running any `bazel` commands to validate. Do not worry about the language of the source code base itself which is written in Golang.

Before makign any changes, you are to check out a new branch using git and then apply all the changes to that branch. Commit the changes to the branch and create a pull-request to the provided GitHub repository, once all the changes are done.

You are to use MCP servers for Git and GitHub, when available.
