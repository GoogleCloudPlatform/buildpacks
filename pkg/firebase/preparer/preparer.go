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
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/envvars"
	secrets "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/secrets"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
)

// Options contains data for the preparer to perform pre-build logic.
type Options struct {
	SecretClient                  secrets.SecretManager
	AppHostingYAMLPath            string
	ProjectID                     string
	AppHostingYAMLOutputFilePath  string
	EnvDereferencedOutputFilePath string
	BackendRootDirectory          string
	BuildpackConfigOutputFilePath string
}

// Prepare performs pre-build logic for App Hosting backends including:
// * Reading, sanitizing, and writing user-defined environment variables in apphosting.yaml to a new file.
// * Dereferencing secrets in apphosting.yaml.
// * Writing the build directory context to files on disk for other build steps to consume.
//
// Preparer will always write a prepared apphosting.yaml and a .env file to disk, even if there
// is no schema to write.
func Prepare(ctx context.Context, opts Options) error {
	dereferencedEnvMap := map[string]string{} // Env map with dereferenced secret material
	appHostingYAML := apphostingschema.AppHostingSchema{}

	if opts.AppHostingYAMLPath != "" {
		var err error

		appHostingYAML, err = apphostingschema.ReadAndValidateAppHostingSchemaFromFile(opts.AppHostingYAMLPath)
		if err != nil {
			return fmt.Errorf("reading in and validating apphosting.yaml at path %v: %w", opts.AppHostingYAMLPath, err)
		}

		apphostingschema.Sanitize(&appHostingYAML)

		if err := secrets.Normalize(appHostingYAML.Env, opts.ProjectID); err != nil {
			return fmt.Errorf("normalizing apphosting.yaml fields: %w", err)
		}

		if err := secrets.PinVersions(ctx, opts.SecretClient, appHostingYAML.Env); err != nil {
			return fmt.Errorf("pinning secrets in apphosting.yaml: %w", err)
		}

		if dereferencedEnvMap, err = secrets.GenerateBuildDereferencedEnvMap(ctx, opts.SecretClient, appHostingYAML.Env); err != nil {
			return fmt.Errorf("dereferencing secrets in apphosting.yaml: %w", err)
		}
	}

	if err := appHostingYAML.WriteToFile(opts.AppHostingYAMLOutputFilePath); err != nil {
		return fmt.Errorf("writing final apphosting.yaml to %v: %w", opts.AppHostingYAMLOutputFilePath, err)
	}

	if err := envvars.Write(dereferencedEnvMap, opts.EnvDereferencedOutputFilePath); err != nil {
		return fmt.Errorf("writing final dereferenced environment variables to %v: %w", opts.EnvDereferencedOutputFilePath, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	if err := util.WriteBuildDirectoryContext(cwd, opts.BackendRootDirectory, opts.BuildpackConfigOutputFilePath); err != nil {
		return fmt.Errorf("writing build directory context to %v: %w", opts.BuildpackConfigOutputFilePath, err)
	}

	return nil
}
