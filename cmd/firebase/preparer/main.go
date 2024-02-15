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
	"flag"
	"log"

	preparer "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/preparer"
	"cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go/v2"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	apphostingEnvFilePath         = flag.String("apphostingenv_filepath", "", "File path to user defined apphosting.env")
	projectID                     = flag.String("project_id", "", "User's GCP project ID")
	envReferencedOutputFilePath   = flag.String("env_referenced_output_filepath", "", "File path to write sanitized environment variables & referenced secret material to")
	envDereferencedOutputFilePath = flag.String("env_dereferenced_output_filepath", "", "File path to write sanitized environment variables & dereferenced secret material to")
)

func main() {
	flag.Parse()

	if *projectID == "" {
		log.Fatal("--project_id flag not specified.")
	}

	if *envReferencedOutputFilePath == "" {
		log.Fatal("--env_referenced_output_filepath flag not specified.")
	}

	if *envDereferencedOutputFilePath == "" {
		log.Fatal("--env_dereferenced_output_filepath flag not specified.")
	}

	err := preparer.Prepare(*apphostingEnvFilePath, *projectID, *envReferencedOutputFilePath, *envDereferencedOutputFilePath)
	if err != nil {
		log.Fatal(err)
	}
}

// TempSecretManager is an interface for the Secret Manager API, used as a temporary holder to manage dependency imports.
type TempSecretManager interface {
	GetSecretVersion(opts ...gax.CallOption) error
}

// TempGetSecretVersionResponse is a wrapper for a temp secret manager service GetSecretVersion api response. Used to temporarily manage dependency imports.
type TempGetSecretVersionResponse struct {
	SecretVersion *smpb.SecretVersion
	SecretClient  *secretmanager.Client
	Error         error
}
