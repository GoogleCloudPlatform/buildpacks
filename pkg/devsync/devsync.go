// Copyright 2026 Google LLC
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

// Package devsync provides common utilities for configuring the runit service tree in DevSync mode.
package devsync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UpdateAppRunScript reads the run.template file, replaces {{ENTRYPOINT}} with the provided webCmd,
// populates the Runit envdir with exported environment variables, and writes the executable app/run script.
func UpdateAppRunScript(serviceDir, webCmd string, envVars map[string]string) error {
	if err := updateEnvDir(serviceDir, envVars); err != nil {
		return err
	}
	if err := renderRunScript(serviceDir, webCmd); err != nil {
		return err
	}
	return nil
}

func updateEnvDir(serviceDir string, envVars map[string]string) error {
	envDir := filepath.Join(serviceDir, "app", "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("creating app/env directory: %w", err)
	}

	for k, v := range envVars {
		envFilePath := filepath.Join(envDir, k)
		if err := os.WriteFile(envFilePath, []byte(v), 0644); err != nil {
			return fmt.Errorf("writing env var %s: %w", k, err)
		}
	}
	return nil
}

func renderRunScript(serviceDir, webCmd string) error {
	templatePath := filepath.Join(serviceDir, "app", "run.template")
	runPath := filepath.Join(serviceDir, "app", "run")

	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("reading run.template: %w", err)
	}
	replaced := strings.ReplaceAll(string(content), "{{ENTRYPOINT}}", webCmd)

	if err := os.WriteFile(runPath, []byte(replaced), 0755); err != nil {
		return fmt.Errorf("writing app/run: %w", err)
	}
	return nil
}
