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

	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	env "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/env"
)

// outputBundleSchema is the struct representation of a Firebase App Hosting Output Bundle
// (configured by bundle.yaml).
type outputBundleSchema struct {
	// TODO: Add fields.
}

// buildSchema is the internal Publisher representation of the final build settings that will
// ultimately be converted into an updateBuildRequest.
type buildSchema struct {
	RunConfig *apphostingschema.RunConfig            `yaml:"runConfig,omitempty"`
	Runtime   *runtime                               `yaml:"runtime,omitempty"`
	Env       []apphostingschema.EnvironmentVariable `yaml:"env,omitempty"`
}

// TODO (b/328444933): Migrate this to the new EnvironmentVariable in apphostingschema.go
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

func toBuildSchema(schema apphostingschema.AppHostingSchema, bundleSchema outputBundleSchema, appHostingEnvVars map[string]string) buildSchema {
	dCPU := float32(defaultCPU)
	buildSchema := buildSchema{
		RunConfig: &apphostingschema.RunConfig{
			CPU:          &dCPU,
			MemoryMiB:    &defaultMemory,
			Concurrency:  &defaultConcurrency,
			MaxInstances: &defaultMaxInstances,
			MinInstances: &defaultMinInstances,
		},
	}
	// Copy fields from apphosting.yaml.
	b := schema.RunConfig
	if b.CPU != nil {
		cpu := float32(*b.CPU)
		buildSchema.RunConfig.CPU = &cpu
	}
	if b.MemoryMiB != nil {
		buildSchema.RunConfig.MemoryMiB = b.MemoryMiB
	}
	if b.Concurrency != nil {
		buildSchema.RunConfig.Concurrency = b.Concurrency
	}
	if b.MaxInstances != nil {
		buildSchema.RunConfig.MaxInstances = b.MaxInstances
	}
	if b.MinInstances != nil {
		buildSchema.RunConfig.MinInstances = b.MinInstances
	}

	// Copy fields from apphosting.env.
	if len(appHostingEnvVars) > 0 {
		buildSchema.Runtime = &runtime{EnvVariables: appHostingEnvVars}
		envVars := []apphostingschema.EnvironmentVariable{}
		for k, v := range appHostingEnvVars {
			ev := apphostingschema.EnvironmentVariable{
				Variable:     k,
				Value:        v,
				Availability: []string{"RUNTIME"},
			}
			envVars = append(envVars, ev)
		}
		buildSchema.Env = envVars
	}
	return buildSchema
}

// Publish takes in the path to various required files such as apphosting.yaml, bundle.yaml, and
// other files (tbd) and merges them into one output that describes the desired Backend Service
// configuration before pushing this information to the control plane.
func Publish(appHostingYAMLPath string, bundleYAMLPath string, envPath string, outputFilePath string) error {
	apphostingYAML, err := apphostingschema.ReadAndValidateAppHostingSchemaFromFile(appHostingYAMLPath)
	if err != nil {
		return err
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
