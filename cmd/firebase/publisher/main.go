// Copyright 2023 Google LLC
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

// The main binary will in the future run AppHosting publisher logic.
package main

import (
	"flag"
	"log"
	"path/filepath"

	publisher "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/publisher"
)

var (
	apphostingYAMLFilePath = flag.String("apphostingyaml_filepath", "", "File path to user defined apphosting.yaml")
	outputBundleDir        = flag.String("output_bundle_dir", "", "File path to root directory of build artifacts aka Output Bundle (including bundle.yaml)")
	outputFilePath         = flag.String("output_filepath", "", "File path to write publisher output data to")
)

func main() {

	flag.Parse()

	if *apphostingYAMLFilePath == "" {
		log.Fatal("--apphostingyaml_filepath flag not specified.")
	}
	if *outputBundleDir == "" {
		log.Fatal("--output_bundle_dir flag not specified.")
	}

	if *outputFilePath == "" {
		log.Fatal("--output_filepath flag not specified.")
	}

	err := publisher.Publish(
		*apphostingYAMLFilePath, filepath.Join(*outputBundleDir, "bundle.yaml"), *outputFilePath)
	if err != nil {
		log.Fatal(err)
	}
}
