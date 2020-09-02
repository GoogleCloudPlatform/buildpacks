// Copyright 2020 Google LLC
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

// The main binary verifies that all tools are correctly installed.
package main

import (
	"log"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/checktools"
)

func main() {
	log.Printf("Checking tools")
	if err := checktools.Installed(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Printf("Checking pack version")
	if err := checktools.PackVersion(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
