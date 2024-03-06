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
	"context"
	"fmt"

	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
	secrets "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/secrets"
)

// Prepare performs pre-build logic for App Hosting backends including:
// * Reading, sanitizing, and writing user-defined environment variables in apphosting.env to a new file.
//
// Prepare will always write a file to disk, even if there are no environment variables to write.
func Prepare(ctx context.Context, secretClient secrets.SecretManager, apphostingEnvFilePath string, projectID string, envReferencedOutputFilePath string, envDereferencedOutputFilePath string) error {
	referencedEnvMap := map[string]string{}   // Env map with referenced secret material
	dereferencedEnvMap := map[string]string{} // Env map with dereferenced secret material

	// Read statically defined env vars from apphosting.env.
	if apphostingEnvFilePath != "" {
		var err error

		referencedEnvMap, err = env.ReadEnv(apphostingEnvFilePath)
		if err != nil {
			return fmt.Errorf("reading apphosting.env: %w", err)
		}
		referencedEnvMap, err = env.SanitizeAppHostingEnv(referencedEnvMap)
		if err != nil {
			return fmt.Errorf("sanitizing apphosting.env fields: %w", err)
		}
		err = secrets.NormalizeAppHostingSecretsEnv(referencedEnvMap, projectID)
		if err != nil {
			return fmt.Errorf("normalizing apphosting.env fields: %w", err)
		}
		err = secrets.PinVersionSecrets(ctx, secretClient, referencedEnvMap)
		if err != nil {
			return fmt.Errorf("pinning secrets in apphosting.env: %w", err)
		}
		dereferencedEnvMap, err = secrets.DereferenceSecrets(ctx, secretClient, referencedEnvMap)
		if err != nil {
			return fmt.Errorf("dereferencing secrets in apphosting.env: %w", err)
		}
	}

	err := env.WriteEnv(referencedEnvMap, envReferencedOutputFilePath)
	if err != nil {
		return fmt.Errorf("writing final referenced environment variables to %v: %w", envReferencedOutputFilePath, err)
	}

	err = env.WriteEnv(dereferencedEnvMap, envDereferencedOutputFilePath)
	if err != nil {
		return fmt.Errorf("writing final dereferenced environment variables to %v: %w", envDereferencedOutputFilePath, err)
	}

	return nil
}
