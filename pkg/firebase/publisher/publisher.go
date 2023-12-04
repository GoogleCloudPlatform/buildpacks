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
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// appHostingSchema is the struct representation of apphosting.yaml
type appHostingSchema struct {
	BackendResources backendResources `yaml:"backend_resources"`
}

// buildSchema is the internal Publisher representation of the final build settings that will ultimately be converted into an updateBuildRequest
type buildSchema struct {
	BackendResources backendResources `yaml:"backend_resources"`
}

type backendResources struct {
	CPU          int32
	Memory       int32
	Concurrency  int32
	MinInstances int32 `yaml:"minInstances"`
	MaxInstances int32 `yaml:"maxInstances"`
}

func byteArrayToAppHostingSchema(fileData []byte) (*appHostingSchema, error) {
	var structData appHostingSchema
	err := yaml.Unmarshal(fileData, &structData)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling %q as YAML: %w", fileData, err)
	}
	return &structData, nil
}

func validateAppHostingYAMLFields(appHostingYAML *appHostingSchema) error {
	if !(appHostingYAML.BackendResources.CPU >= 1 && appHostingYAML.BackendResources.CPU <= 8) {
		return fmt.Errorf("cpu field is not in valid range of '1 <= cpu <= 8'")
	}

	if !(appHostingYAML.BackendResources.Memory >= 512 && appHostingYAML.BackendResources.Memory <= 32768) {
		return fmt.Errorf("memory field is not in valid range of '512 <= memory <= 32768'")
	}

	if !(appHostingYAML.BackendResources.Concurrency >= 1 && appHostingYAML.BackendResources.Concurrency <= 1000) {
		return fmt.Errorf("concurrency field is not in valid range of '1 <= concurrency <= 1000'")
	}

	if !(appHostingYAML.BackendResources.MinInstances >= 1 && appHostingYAML.BackendResources.MinInstances <= 1000) {
		return fmt.Errorf("minInstances field is not in valid range of '1 <= minInstances <= 1000'")
	}

	if !(appHostingYAML.BackendResources.MaxInstances >= 1 && appHostingYAML.BackendResources.MaxInstances <= 100) {
		return fmt.Errorf("maxInstances field is not in valid range of '1 <= maxInstances <= 100'")
	}

	return nil
}

// Write the given build schema to the specified path, used to output the final arguments to BuildStepOutputs[]
func writeToFile(fileData []byte, outputFilePath string) error {
	err := os.MkdirAll(filepath.Dir(outputFilePath), os.ModeDir)
	if err != nil {
		return fmt.Errorf("creating parent directory %q: %w", outputFilePath, err)
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(fileData)
	if err != nil {
		return fmt.Errorf("writing data to file: %w", err)
	}

	return nil
}

// Publish takes in the path to various required files such as apphosting.yaml, bundle.yaml, and other files (tbd) and merges them into one output that describes the
// desired Backend Service configuration before pushing this information to the control plane.
func Publish(appHostingYAMLPath string, outputFilePath string) error {
	// Read in apphosting.yaml
	apphostingBuffer, err := os.ReadFile(appHostingYAMLPath)
	if err != nil {
		return err
	}

	// Convert read in file arrays to struct
	apphostingYAML, err := byteArrayToAppHostingSchema(apphostingBuffer)
	if err != nil {
		return err
	}

	validateErr := validateAppHostingYAMLFields(apphostingYAML)
	if validateErr != nil {
		return fmt.Errorf("apphosting.yaml fields are not valid: %w", validateErr)
	}

	// TODO: Use bundleYaml and apphostingYaml to generate output.yaml
	buildSchema := buildSchema{
		BackendResources: backendResources{
			CPU:          apphostingYAML.BackendResources.CPU,
			Memory:       apphostingYAML.BackendResources.Memory,
			Concurrency:  apphostingYAML.BackendResources.Concurrency,
			MinInstances: apphostingYAML.BackendResources.MinInstances,
			MaxInstances: apphostingYAML.BackendResources.MaxInstances,
		},
	}

	// Marshal struct to YAML format.
	buildSchemaData, err := yaml.Marshal(&buildSchema)
	if err != nil {
		return fmt.Errorf("converting struct to YAML: %w", err)
	}

	err = writeToFile(buildSchemaData, outputFilePath)
	if err != nil {
		return fmt.Errorf("writing buildSchema to file: %w", err)
	}

	return nil
}
