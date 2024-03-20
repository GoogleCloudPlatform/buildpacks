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
	"flag"
	"fmt"
	"log"

	preparer "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/preparer"
	"cloud.google.com/go/secretmanager/apiv1"
)

var (
	apphostingYAMLFilePath       = flag.String("apphostingyaml_filepath", "", "File path to user defined apphosting.yaml")
	projectID                    = flag.String("project_id", "", "User's GCP project ID")
	appHostingYAMLOutputFilePath = flag.String("apphostingyaml_output_filepath", "", "File path to write the validated and formatted apphosting.yaml to")
	dotEnvOutputFilePath         = flag.String("dot_env_output_filepath", "", "File path to write the output .env file to")
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

	secretClient, err := secretmanager.NewClient(context.Background())
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create secretmanager client: %w", err))
	}
	defer secretClient.Close()

	err = preparer.Prepare(context.Background(), secretClient, *apphostingYAMLFilePath, *projectID, *appHostingYAMLOutputFilePath, *dotEnvOutputFilePath)
	if err != nil {
		log.Fatal(err)
	}
}
