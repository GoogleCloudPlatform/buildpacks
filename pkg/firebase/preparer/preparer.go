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
	"slices"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/envvars"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/secrets"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
)

// Options contains data for the preparer to perform pre-build logic.
type Options struct {
	SecretClient                      secrets.SecretManager
	AppHostingYAMLPath                string
	ProjectID                         string
	Region                            string
	EnvironmentName                   string
	AppHostingYAMLOutputFilePath      string
	EnvDereferencedOutputFilePath     string
	BackendRootDirectory              string
	BuildpackConfigOutputFilePath     string
	FirebaseConfig                    string
	FirebaseWebappConfig              string
	ServerSideEnvVars                 string
	ApphostingPreprocessedPathForPack string
}

const ngTrustProxyHeaders = "NG_TRUST_PROXY_HEADERS"
const expectedProxyHeaderValue = "X-Forwarded-Host"

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
	var err error

	if opts.AppHostingYAMLPath != "" {
		appHostingYAML, err = apphostingschema.ReadAndValidateFromFile(opts.AppHostingYAMLPath)
		if err != nil {
			return fmt.Errorf("reading in and validating apphosting.yaml at path %v: %w", opts.AppHostingYAMLPath, err)
		}
		for i := range appHostingYAML.Env {
			appHostingYAML.Env[i].Source = apphostingschema.SourceAppHostingYAML
		}

		if err = apphostingschema.MergeWithEnvironmentSpecificYAML(&appHostingYAML, opts.AppHostingYAMLPath, opts.EnvironmentName); err != nil {
			return fmt.Errorf("merging with environment specific apphosting.%v.yaml: %w", opts.EnvironmentName, err)
		}
	}

	// Add FIREBASE_CONFIG env var for Admin SDK AutoInit, only if it is not already user-defined.
	if _, found := apphostingschema.GetEnvVar(&appHostingYAML, "FIREBASE_CONFIG"); !found {
		if opts.FirebaseConfig != "" {
			appHostingYAML.Env = append(appHostingYAML.Env, apphostingschema.EnvironmentVariable{Variable: "FIREBASE_CONFIG", Value: opts.FirebaseConfig, Source: apphostingschema.SourceFirebaseSystem})
		}
	}

	// Add FIREBASE_WEBAPP_CONFIG env var for Client SDK AutoInit, only if it is not already user-defined.
	if _, found := apphostingschema.GetEnvVar(&appHostingYAML, "FIREBASE_WEBAPP_CONFIG"); !found {
		if opts.FirebaseWebappConfig != "" {
			appHostingYAML.Env = append(appHostingYAML.Env, apphostingschema.EnvironmentVariable{Variable: "FIREBASE_WEBAPP_CONFIG", Value: opts.FirebaseWebappConfig, Availability: []string{"BUILD"}, Source: apphostingschema.SourceFirebaseSystem})
		}
	}

	// Merge server side env vars, overriding any values defined in other env var sources.
	if opts.ServerSideEnvVars != "" {
		parsedServerSideEnvVars, err := envvars.ParseEnvVarsFromString(opts.ServerSideEnvVars)
		if err != nil {
			return fmt.Errorf("parsing server side env vars %v: %w", opts.ServerSideEnvVars, err)
		}

		appHostingYAML.Env = apphostingschema.MergeEnvVars(appHostingYAML.Env, parsedServerSideEnvVars)
	}

	// NG_TRUST_PROXY_HEADERS is used by Angular Universal applications to determine which proxy headers to trust.
	// We inject it here so that it is available to users by default. If the user provides this env var,
	// we validate its value and use it if correct.
	if ev, found := apphostingschema.GetEnvVar(&appHostingYAML, ngTrustProxyHeaders); found {
		// Validate the value of the user provided NG_TRUST_PROXY_HEADERS
		if ev.Value != expectedProxyHeaderValue {
			return fmt.Errorf("invalid value for %s (Angular trust proxy headers): got %q, want %q", ngTrustProxyHeaders, ev.Value, expectedProxyHeaderValue)
		}

		if len(ev.Availability) > 0 && !slices.Contains(ev.Availability, "RUNTIME") {
			return fmt.Errorf("user-defined environment variable %s must include RUNTIME in its availability", ngTrustProxyHeaders)
		}
	} else {
		appHostingYAML.Env = append(appHostingYAML.Env, apphostingschema.EnvironmentVariable{
			Variable:     ngTrustProxyHeaders,
			Value:        expectedProxyHeaderValue,
			Source:       apphostingschema.SourceFirebaseSystem,
			Availability: []string{"RUNTIME"},
		})
	}

	// NG_ALLOWED_HOSTS secures Angular SSR against Host Poisoning. We try to derive it
	// from X_FIREBASE_SUPPORTED_HOSTS (reserved default domains provided by Firebase) if not user-set.
	// Skip silently if X_FIREBASE_SUPPORTED_HOSTS is empty or missing.
	if _, found := apphostingschema.GetEnvVar(&appHostingYAML, "NG_ALLOWED_HOSTS"); !found {
		ev, _ := apphostingschema.GetEnvVar(&appHostingYAML, "X_FIREBASE_SUPPORTED_HOSTS")
		supportedHosts := ev.Value

		if supportedHosts != "" {
			// Derived NG_ALLOWED_HOSTS defaults to RUNTIME availability as it is required at runtime.
			appHostingYAML.Env = append(appHostingYAML.Env, apphostingschema.EnvironmentVariable{
				Variable:     "NG_ALLOWED_HOSTS",
				Value:        supportedHosts,
				Availability: []string{"RUNTIME"},
				Source:       apphostingschema.SourceFirebaseSystem,
			})
		}
	}
	apphostingschema.Sanitize(&appHostingYAML)

	// If NG_ALLOWED_HOSTS exists at this point, ensure it has RUNTIME availability.
	if ev, found := apphostingschema.GetEnvVar(&appHostingYAML, "NG_ALLOWED_HOSTS"); found {
		if !slices.Contains(ev.Availability, "RUNTIME") {
			return fmt.Errorf("NG_ALLOWED_HOSTS environment variable must be set with RUNTIME availability")
		}
	}

	if err := secrets.Normalize(appHostingYAML.Env, opts.ProjectID); err != nil {
		return fmt.Errorf("normalizing apphosting.yaml fields: %w", err)
	}

	if err := secrets.PinVersions(ctx, opts.SecretClient, appHostingYAML.Env); err != nil {
		return fmt.Errorf("pinning secrets in apphosting.yaml: %w", err)
	}

	if dereferencedEnvMap, err = secrets.GenerateBuildDereferencedEnvMap(ctx, opts.SecretClient, appHostingYAML.Env); err != nil {
		return fmt.Errorf("dereferencing secrets in apphosting.yaml: %w", err)
	}

	apphostingschema.NormalizeVpcAccess(appHostingYAML.RunConfig.VpcAccess, opts.ProjectID, opts.Region)

	if err := appHostingYAML.WriteToFile(opts.AppHostingYAMLOutputFilePath); err != nil {
		return fmt.Errorf("writing final apphosting.yaml to %v: %w", opts.AppHostingYAMLOutputFilePath, err)
	}

	// The processed apphosting.yaml needs to be written to ApphostingPreprocessedPathForPack since the pack command cannot read from volumes (/yaml in this case)
	if err := appHostingYAML.WriteToFile(opts.ApphostingPreprocessedPathForPack); err != nil {
		return fmt.Errorf("writing final apphosting.yaml to %v: %w", opts.ApphostingPreprocessedPathForPack, err)
	}

	if err := envvars.WriteLifecycle(dereferencedEnvMap, opts.EnvDereferencedOutputFilePath); err != nil {
		return fmt.Errorf("writing final dereferenced environment variables to %v: %w", opts.EnvDereferencedOutputFilePath, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	if err := util.WriteBuildDirectoryContext(cwd, opts.BackendRootDirectory, opts.BuildpackConfigOutputFilePath); err != nil {
		return fmt.Errorf("writing build directory context: %w", err)
	}

	return nil
}
