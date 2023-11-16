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

// Package publisher provides basic functionality to coalesce user and framework adapter defined
// variables. In the future it will provide more functionality.
package publisher

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type backendResources struct {
	CPU int64
}

func byteArrayToBackendResources(fileData []byte) (*backendResources, error) {
	var structData backendResources
	err := yaml.Unmarshal(fileData, &structData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling %q as YAML: %w", fileData, err)
	}
	return &structData, nil
}

// Publish takes in the path to various required files such as apphosting.yaml, bundle.yaml, and other files (tbd) and merges them into one output that describes the
// desired cloud run variables before pushing this information to the AppHosting control plane.
func Publish(pathToAppHostingYAML, pathToBundleYAML string) (string, error) {
	// Read in bundle.yaml
	bundleBuffer, err := os.ReadFile(pathToBundleYAML)
	if err != nil {
		return "", err
	}

	// Read in apphosting.yaml
	apphostingBuffer, err := os.ReadFile(pathToAppHostingYAML)
	if err != nil {
		return "", err
	}

	// Convert read in file arrays to struct
	bundleYAML, _ := byteArrayToBackendResources(bundleBuffer)
	apphostingYAML, _ := byteArrayToBackendResources(apphostingBuffer)

	// Use bundleYaml and apphostingYaml to generate output.yaml
	bundleYAMLCPU := bundleYAML.CPU
	appHostingYAMLCPU := apphostingYAML.CPU

	outputYAMLCPU := max(bundleYAMLCPU, appHostingYAMLCPU, 1)

	outputYAMLStruct := backendResources{
		CPU: outputYAMLCPU,
	}

	// Marshal struct to YAML format.
	stringOutputYAML, err := yaml.Marshal(&outputYAMLStruct)
	if err != nil {
		return "", fmt.Errorf("error when converting struct to YAML: %w", err)
	}

	// Return string representation of output YAML
	return string(stringOutputYAML), nil
}
