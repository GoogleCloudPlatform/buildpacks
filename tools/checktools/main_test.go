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

// This test verifies that the required tools are correctly set up on the
// host machine rather than testing the checktools package. Blaze/bazel test
// does not preserve the user PATH settings for reproducibility which means
// that the required tools need to be available on PATH as set during the test.
package main

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/checktools"
)

func TestInstalled(t *testing.T) {
	if err := checktools.Installed(); err != nil {
		t.Fatalf("Checking tools: %v", err)
	}
}

func TestPackVersion(t *testing.T) {
	if err := checktools.PackVersion(); err != nil {
		t.Fatalf("Checking pack version: %v", err)
	}
}
