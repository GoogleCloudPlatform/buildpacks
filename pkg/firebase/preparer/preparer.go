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

// Package preparer prepares user-defined environment variables for use in subsequent buildpacks
// steps.
package preparer

import (
	"fmt"

	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
)

// Prepare performs pre-build logic for App Hosting backends including:
// * Reading, sanitizing, and writing user-defined environment variables in apphosting.env to a new file.
//
// Prepare will always write a file to disk, even if there are no environment variables to write.
func Prepare(apphostingEnvFilePath string, envOutputFilePath string) error {
	envMap := map[string]string{}

	// Read statically defined env vars from apphosting.env.
	if apphostingEnvFilePath != "" {
		var err error
		envMap, err = env.ReadEnv(apphostingEnvFilePath)
		if err != nil {
			return fmt.Errorf("reading apphosting.env: %w", err)
		}
		envMap, err = env.SanitizeAppHostingEnv(envMap)
		if err != nil {
			return fmt.Errorf("sanitizing apphosting.env fields: %w", err)
		}
	}

	err := env.WriteEnv(envMap, envOutputFilePath)
	if err != nil {
		return fmt.Errorf("writing final environment variables to %v: %w", envOutputFilePath, err)
	}

	return nil
}
