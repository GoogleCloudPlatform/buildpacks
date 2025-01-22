// Copyright 2025 Google LLC
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

// Package filesystem provides utility functions to navigate user's repository.
package filesystem

import (
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
)

var (
	appHostingYamlRegex = regexp.MustCompile(`^apphosting(\.[a-z0-9_]+)?\.yaml$`)
)

// DetectAppHostingYAMLPath returns the absolute path to the apphosting.yaml file, or an error if no
// such file is found.
func DetectAppHostingYAMLPath(workspacePath, backendRootDirectory string) (string, error) {
	backendRootDirectoryAbsolutePath := path.Join(workspacePath, backendRootDirectory)
	_, err := os.Stat(backendRootDirectoryAbsolutePath)
	if err != nil {
		return "", faherror.InvalidRootDirectoryError(backendRootDirectoryAbsolutePath, err)
	}

	apphostingYamlRoot, err := detectAppHostingYAMLRoot(backendRootDirectoryAbsolutePath)
	if err != nil {
		return "", err
	}

	return path.Join(apphostingYamlRoot, "apphosting.yaml"), nil
}

// detectAppHostingYAMLRoot returns the root directory containing the apphosting.yaml file, or an
// error if no such file is found.
//
// If no apphosting.yaml file is found in the root directory or any parent directories an empty
// string is returned.
//
// "root" is an absolute path.
func detectAppHostingYAMLRoot(root string) (string, error) {
	current := root
	for {
		appHostingFileExists, err := appHostingYAMLFileExistsInDir(current)
		if err != nil {
			return "", fmt.Errorf("failed to check for apphosting.yaml root directory: %w", err)
		}

		if appHostingFileExists {
			return current, nil
		}

		parentDir := path.Dir(current)
		if parentDir == current {
			return "", nil
		}

		current = parentDir
	}
}

func appHostingYAMLFileExistsInDir(dir string) (bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, f := range files {
		if appHostingYamlRegex.MatchString(f.Name()) {
			return true, nil
		}
	}
	return false, nil
}
