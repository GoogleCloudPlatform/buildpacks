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

// The preparer binary runs preprocessing steps for App Hosting backend builds.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/preparer"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"cloud.google.com/go/secretmanager/apiv1"
)

var (
	apphostingYAMLFilePath        = flag.String("apphostingyaml_filepath", "", "File path to user defined apphosting.yaml")
	projectID                     = flag.String("project_id", "", "User's GCP project ID")
	environmentName               = flag.String("environment_name", "", "Environment name tied to the build, if applicable")
	appHostingYAMLOutputFilePath  = flag.String("apphostingyaml_output_filepath", "", "File path to write the validated and formatted apphosting.yaml to")
	dotEnvOutputFilePath          = flag.String("dot_env_output_filepath", "", "File path to write the output .env file to")
	backendRootDirectory          = flag.String("backend_root_directory", "", "File path to the application directory specified by the user")
	buildpackConfigOutputFilePath = flag.String("buildpack_config_output_filepath", "", "File path to write the buildpack config to")
	firebaseConfig                = flag.String("firebase_config", "", "JSON serialized Firebase config used by Firebase Admin SDK")
	serverSideEnvVars             = flag.String("server_side_env_vars", "", "List of server side env vars to set. An empty string indicates server side environment variables are disabled. Any other value indicates enablement and to use these vars over yaml defined env vars.")
)

func main() {
	flag.Parse()

	if *projectID == "" {
		log.Fatal("--project_id flag not specified.")
	}

	if *appHostingYAMLOutputFilePath == "" {
		log.Fatal("--apphostingyaml_output_filepath flag not specified.")
	}

	if *dotEnvOutputFilePath == "" {
		log.Fatal("--dot_env_output_filepath flag not specified.")
	}

	if backendRootDirectory == nil {
		log.Fatal("--backend_root_directory flag not specified.")
	}

	if *buildpackConfigOutputFilePath == "" {
		log.Fatal("--buildpack_config_output_filepath flag not specified.")
	}

	secretClient, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create secretmanager client: %w", err))
	}
	defer secretClient.Close()

	opts := preparer.Options{
		SecretClient:                  secretClient,
		AppHostingYAMLPath:            *apphostingYAMLFilePath,
		ProjectID:                     *projectID,
		EnvironmentName:               *environmentName,
		AppHostingYAMLOutputFilePath:  *appHostingYAMLOutputFilePath,
		EnvDereferencedOutputFilePath: *dotEnvOutputFilePath,
		BackendRootDirectory:          *backendRootDirectory,
		BuildpackConfigOutputFilePath: *buildpackConfigOutputFilePath,
		FirebaseConfig:                *firebaseConfig,
		ServerSideEnvVars:             *serverSideEnvVars,
	}

	gcpCtx := gcpbuildpack.NewContext()

	if err = preparer.Prepare(context.Background(), opts); err != nil {
		var fe *faherror.FahError
		if errors.As(err, &fe) {
			// Known App Hosting user errors are wrapped by a GCP User Error to avoid being labeled
			// as internal status errors.
			err = gcpbuildpack.UserErrorf("%w", fe)
		} else {
			err = faherror.InternalErrorf("%w", err)
		}
		gcpCtx.Exit(1, err)
	}
	gcpCtx.Exit(0, nil)
}
