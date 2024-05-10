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

// Package preparer prepares user-defined apphosting.yaml for use in subsequent buildpacks
// steps.
package preparer

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/envvars"

	secrets "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/secrets"
)

// Prepare performs pre-build logic for App Hosting backends including:
// * Reading, sanitizing, and writing user-defined environment variables in apphosting.yaml to a new file.
// * Dereferencing secrets in apphosting.yaml.
//
// Preparer will always write a prepared apphosting.yaml and a .env file to disk, even if there
// is no schema to write.
func Prepare(ctx context.Context, secretClient secrets.SecretManager, appHostingYAMLPath string, projectID string, appHostingYAMLOutputFilePath string, envDereferencedOutputFilePath string) error {
	dereferencedEnvMap := map[string]string{} // Env map with dereferenced secret material
	appHostingYAML := apphostingschema.AppHostingSchema{}

	if appHostingYAMLPath != "" {
		var err error

		appHostingYAML, err = apphostingschema.ReadAndValidateAppHostingSchemaFromFile(appHostingYAMLPath)
		if err != nil {
			return fmt.Errorf("reading in and validating apphosting.yaml at path %v: %w", appHostingYAMLPath, err)
		}

		apphostingschema.Sanitize(&appHostingYAML)

		err = secrets.Normalize(appHostingYAML.Env, projectID)
		if err != nil {
			return fmt.Errorf("normalizing apphosting.yaml fields: %w", err)
		}

		err = secrets.PinVersions(ctx, secretClient, appHostingYAML.Env)
		if err != nil {
			return fmt.Errorf("pinning secrets in apphosting.yaml: %w", err)
		}

		dereferencedEnvMap, err = secrets.GenerateBuildDereferencedEnvMap(ctx, secretClient, appHostingYAML.Env)
		if err != nil {
			return fmt.Errorf("dereferencing secrets in apphosting.yaml: %w", err)
		}
	}

	err := appHostingYAML.WriteToFile(appHostingYAMLOutputFilePath)
	if err != nil {
		return fmt.Errorf("writing final apphosting.yaml to %v: %w", appHostingYAMLOutputFilePath, err)
	}

	err = envvars.Write(dereferencedEnvMap, envDereferencedOutputFilePath)
	if err != nil {
		return fmt.Errorf("writing final dereferenced environment variables to %v: %w", envDereferencedOutputFilePath, err)
	}

	return nil
}
