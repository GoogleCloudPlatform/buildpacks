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
)

var (
	apphostingEnvFilePath = flag.String("apphostingenv_filepath", "", "File path to user defined apphosting.env")
	envOutputFilePath     = flag.String("env_output_filepath", "", "File path to write sanitized environment variables too")
)

func main() {
	flag.Parse()

	if *envOutputFilePath == "" {
		log.Fatal("--env_output_filepath flag not specified.")
	}

	err := preparer.Prepare(*apphostingEnvFilePath, *envOutputFilePath)
	if err != nil {
		log.Fatal(err)
	}
}
