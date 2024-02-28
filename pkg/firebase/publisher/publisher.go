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
// variables.
package publisher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
)

// appHostingSchema is the struct representation of apphosting.yaml
type appHostingSchema struct {
	BackendResources backendResources `yaml:"backendResources,omitempty"`
}

// outputBundleSchema is the struct representation of a Firebase App Hosting Output Bundle
// (configured by bundle.yaml).
type outputBundleSchema struct {
	// TODO: Add fields.
}

// buildSchema is the internal Publisher representation of the final build settings that will
// ultimately be converted into an updateBuildRequest.
// TODO: b/322201485 - remove the "backendResources" struct once "runConfig" has rolled out.
type buildSchema struct {
	BackendResources backendResources `yaml:"backendResources,omitempty"`
	RunConfig        *runConfig       `yaml:"runConfig,omitempty"`
	Runtime          *runtime         `yaml:"runtime,omitempty"`
}

type backendResources struct {
	// int32 value type used here to match server field types. pointers are used to capture unset vs zero-like values.
	CPU          *int32 `yaml:"cpu"`
	MemoryMiB    *int32 `yaml:"memoryMiB"`
	Concurrency  *int32 `yaml:"concurrency"`
	MaxInstances *int32 `yaml:"maxInstances"`
}

type runConfig struct {
	// value types used must match server field types. pointers are used to capture unset vs zero-like values.
	CPU          *float32 `yaml:"cpu"`
	MemoryMiB    *int32   `yaml:"memoryMiB"`
	Concurrency  *int32   `yaml:"concurrency"`
	MaxInstances *int32   `yaml:"maxInstances"`
	MinInstances *int32   `yaml:"minInstances"`
}

type runtime struct {
	EnvVariables map[string]string `yaml:"envVariables,omitempty"`
}

var (
	defaultCPU          int32 = 1   // From https://cloud.google.com/run/docs/configuring/services/cpu.
	defaultMemory       int32 = 512 // From https://cloud.google.com/run/docs/configuring/services/memory-limits.
	defaultConcurrency  int32 = 80  // From https://cloud.google.com/run/docs/about-concurrency.
	defaultMaxInstances int32 = 100 // From https://cloud.google.com/run/docs/configuring/max-instances.
	defaultMinInstances int32 = 0   // From https://cloud.google.com/run/docs/configuring/min-instances.

	reservedKeys = map[string]bool{
		"PORT":            true,
		"K_SERVICE":       true,
		"K_REVISION":      true,
		"K_CONFIGURATION": true,
	}

	reservedFirebaseKey = "FIREBASE_"
)

// readAppHostingSchemaFromFile returns nil if apphosting.yaml does not exist.
func readAppHostingSchemaFromFile(filePath string) (appHostingSchema, error) {
	a := appHostingSchema{}
	apphostingBuffer, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		log.Printf("Missing apphosting config at %v, using reasonable defaults\n", filePath)
		return a, nil
	} else if err != nil {
		return a, fmt.Errorf("reading apphosting config at %v: %w", filePath, err)
	}

	err = yaml.Unmarshal(apphostingBuffer, &a)
	if err != nil {
		return a, fmt.Errorf("unmarshalling apphosting config as YAML: %w", err)
	}
	return a, nil
}

func validateAppHostingYAMLFields(appHostingYAML appHostingSchema) error {
	b := appHostingYAML.BackendResources
	if b.CPU != nil && !(1 <= *b.CPU && *b.CPU <= 8) {
		return fmt.Errorf("backend_resources.cpu field is not in valid range of [1, 8]")
	}

	if b.MemoryMiB != nil && !(512 <= *b.MemoryMiB && *b.MemoryMiB <= 32768) {
		return fmt.Errorf("backend_resources.memory field is not in valid range of [512, 32768]")
	}

	if b.Concurrency != nil && !(1 <= *b.Concurrency && *b.Concurrency <= 1000) {
		return fmt.Errorf("backend_resources.concurrency field is not in valid range of [1, 1000]")
	}

	if b.MaxInstances != nil && !(1 <= *b.MaxInstances && *b.MaxInstances <= 100) {
		return fmt.Errorf("backend_resources.maxInstances field is not in valid range of [1, 100]")
	}

	// TODO: b/322201485 - Add validation for min_instances once "runConfig" has rolled out.
	return nil
}

func readBundleSchemaFromFile(filePath string) (outputBundleSchema, error) {
	bundleBuffer, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return outputBundleSchema{}, fmt.Errorf("missing bundle config at %v", filePath)
	} else if err != nil {
		return outputBundleSchema{}, fmt.Errorf("reading bundle config at %v: %w", filePath, err)
	}

	err = yaml.Unmarshal(bundleBuffer, &outputBundleSchema{})
	if err != nil {
		return outputBundleSchema{}, fmt.Errorf("unmarshalling bundle config as YAML: %w", err)
	}
	return outputBundleSchema{}, nil
}

// Write the given build schema to the specified path, used to output the final arguments to BuildStepOutputs[]
func writeToFile(buildSchema buildSchema, outputFilePath string) error {
	fileData, err := yaml.Marshal(&buildSchema)
	if err != nil {
		return fmt.Errorf("converting struct to YAML: %w", err)
	}
	log.Printf("Final build schema:\n%v\n", string(fileData))

	err = os.MkdirAll(filepath.Dir(outputFilePath), os.ModeDir)
	if err != nil {
		return fmt.Errorf("creating parent directory %q: %w", outputFilePath, err)
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("creating build schema file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(fileData)
	if err != nil {
		return fmt.Errorf("writing build schema data to file: %w", err)
	}

	return nil
}

func toBuildSchema(appHostingSchema appHostingSchema, bundleSchema outputBundleSchema, appHostingEnvVars map[string]string) buildSchema {
	dCPU := float32(defaultCPU)
	buildSchema := buildSchema{
		BackendResources: backendResources{
			CPU:          &defaultCPU,
			MemoryMiB:    &defaultMemory,
			Concurrency:  &defaultConcurrency,
			MaxInstances: &defaultMaxInstances,
		},
		RunConfig: &runConfig{
			CPU:          &dCPU,
			MemoryMiB:    &defaultMemory,
			Concurrency:  &defaultConcurrency,
			MaxInstances: &defaultMaxInstances,
			MinInstances: &defaultMinInstances,
		},
	}
	// Copy fields from apphosting.yaml.
	b := appHostingSchema.BackendResources
	// TODO: b/322201485 - Add min_instances once "runConfig" has rolled out.
	if b.CPU != nil {
		buildSchema.BackendResources.CPU = b.CPU
		cpu := float32(*b.CPU)
		buildSchema.RunConfig.CPU = &cpu
	}
	if b.MemoryMiB != nil {
		buildSchema.BackendResources.MemoryMiB = b.MemoryMiB
		buildSchema.RunConfig.MemoryMiB = b.MemoryMiB
	}
	if b.Concurrency != nil {
		buildSchema.BackendResources.Concurrency = b.Concurrency
		buildSchema.RunConfig.Concurrency = b.Concurrency
	}
	if b.MaxInstances != nil {
		buildSchema.BackendResources.MaxInstances = b.MaxInstances
		buildSchema.RunConfig.MaxInstances = b.MaxInstances
	}

	// Copy fields from apphosting.env.
	if len(appHostingEnvVars) > 0 {
		buildSchema.Runtime = &runtime{EnvVariables: appHostingEnvVars}
	}
	return buildSchema
}

// Publish takes in the path to various required files such as apphosting.yaml, bundle.yaml, and
// other files (tbd) and merges them into one output that describes the desired Backend Service
// configuration before pushing this information to the control plane.
func Publish(appHostingYAMLPath string, bundleYAMLPath string, envPath string, outputFilePath string) error {
	apphostingYAML, err := readAppHostingSchemaFromFile(appHostingYAMLPath)
	if err != nil {
		return err
	}

	validateErr := validateAppHostingYAMLFields(apphostingYAML)
	if validateErr != nil {
		return fmt.Errorf("apphosting.yaml fields are not valid: %w", validateErr)
	}

	// Read in environment variables
	envMap, err := env.ReadEnv(envPath)
	if err != nil {
		return fmt.Errorf("reading environment variables from %v: %w", envPath, err)
	}

	// For now, simply validates that bundle.yaml exists.
	bundleSchema, err := readBundleSchemaFromFile(bundleYAMLPath)
	if err != nil {
		return err
	}

	buildSchema := toBuildSchema(apphostingYAML, bundleSchema, envMap)

	err = writeToFile(buildSchema, outputFilePath)
	if err != nil {
		return err
	}

	return nil
}
