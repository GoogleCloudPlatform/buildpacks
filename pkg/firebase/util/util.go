// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util provides utility functions to build applications using the Firebase App Hosting builder.
package util

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	supportedMonorepoConfigFiles = []string{"nx.json", "turbo.json"}
)

// ApplicationDirectory looks up the path to the application directory from the environment. Returns
// the application root by default.
func ApplicationDirectory(ctx *gcp.Context) string {
	appDir := ctx.ApplicationRoot()
	if appDirEnv, exists := os.LookupEnv(env.Buildable); exists {
		appDir = filepath.Join(ctx.ApplicationRoot(), appDirEnv)
	}
	return appDir
}

// supportedMonorepoConfigFileExists checks if a supported monorepo config file exists in the
// specified directory.
func supportedMonorepoConfigFileExists(dir string) (bool, error) {
	for _, filename := range supportedMonorepoConfigFiles {
		f := filepath.Join(dir, filename)
		_, err := os.ReadFile(f)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// buildDirectoryContext returns (1) the "build directory" from which the buildpacks will be run,
// and (2) the directory containing the application to be built, relative to the build directory.
//
// The build directory and application directory are different in monorepo contexts, in which we
// want to run the buildpacks process from the root of the monorepo to ensure all necessary files
// are accessible, but we want to build the application inside the user-specified subdirectory.
// see go/apphosting-monorepo-support for more details.
func buildDirectoryContext(cwd, userSpecifiedAppDirPath string) (string, string, error) {
	if userSpecifiedAppDirPath == "" {
		return "", "", nil
	}

	absoluteAppDirPath := filepath.Join(cwd, userSpecifiedAppDirPath)
	_, err := os.Stat(absoluteAppDirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", faherror.InvalidRootDirectoryError(userSpecifiedAppDirPath, err)
		}
		return "", "", err
	}
	var monorepoRootPath string
	curr := absoluteAppDirPath
	for {
		exists, err := supportedMonorepoConfigFileExists(curr)
		if err != nil {
			return "", "", err
		}
		if exists {
			monorepoRootPath = curr
			break
		}
		if curr == cwd || curr == "/" || curr == "." {
			break
		}
		curr = filepath.Dir(curr)
	}
	if monorepoRootPath == "" {
		// If no monorepo config file is detected, then the user-specified app directory path is the
		// root of an application in a subdirectory.
		return userSpecifiedAppDirPath, "", nil
	}
	// If a monorepo config file is detected, then the monorepo root is the "build directory" and the
	// user-specified app directory path is the root of the sub-application.
	mrp, err := filepath.Rel(cwd, monorepoRootPath)
	if err != nil {
		return "", "", err
	}
	adp, err := filepath.Rel(monorepoRootPath, absoluteAppDirPath)
	if err != nil {
		return "", "", err
	}
	return mrp, adp, nil
}

// WriteBuildDirectoryContext writes the build directory context to the specified buildpack config
// file path.
func WriteBuildDirectoryContext(cwd, appDirectoryPath, buildpackConfigOutputFilePath string) error {
	buildDirectory, relativeProjectDirectory, err := buildDirectoryContext(cwd, appDirectoryPath)
	if err != nil {
		return err
	}
	err = os.MkdirAll(buildpackConfigOutputFilePath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(buildpackConfigOutputFilePath, "build-directory.txt"), []byte(buildDirectory), 0644)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(buildpackConfigOutputFilePath, "relative-project-directory.txt"), []byte(relativeProjectDirectory), 0644)
}
