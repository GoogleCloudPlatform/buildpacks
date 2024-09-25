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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/bundleschema"
)

// buildSchema is the internal Publisher representation of the final build settings that will
// ultimately be converted into an updateBuildRequest.
type buildSchema struct {
	RunConfig *apphostingschema.RunConfig            `yaml:"runConfig,omitempty"`
	Env       []apphostingschema.EnvironmentVariable `yaml:"env,omitempty"`
	Metadata  *bundleschema.Metadata                 `yaml:"metadata,omitempty"`
}

var (
	defaultCPU          int32 = 1   // From https://cloud.google.com/run/docs/configuring/services/cpu.
	defaultMemory       int32 = 512 // From https://cloud.google.com/run/docs/configuring/services/memory-limits.
	defaultConcurrency  int32 = 80  // From https://cloud.google.com/run/docs/about-concurrency.
	defaultMaxInstances int32 = 100 // From https://cloud.google.com/run/docs/configuring/max-instances.
	defaultMinInstances int32 = 0   // From https://cloud.google.com/run/docs/configuring/min-instances.
)

// Write the given build schema to the specified path, used to output the final arguments to BuildStepOutputs[]
func writeToFile(buildSchema buildSchema, outputFilePath string) error {
	fileData, err := yaml.Marshal(&buildSchema)
	if err != nil {
		return fmt.Errorf("converting struct to YAML: %w", err)
	}
	log.Printf("Final build schema:\n%v\n. Note that any unset runConfig fields will be set to reasonable default values.", string(fileData))

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

func toBuildSchema(appHostingSchema apphostingschema.AppHostingSchema, bundleSchema bundleschema.BundleSchema) buildSchema {
	buildSchema := buildSchema{}

	// Copy RunConfig fields from apphosting.yaml, Control Plane will set defaults for any unset fields.
	buildSchema.RunConfig = &appHostingSchema.RunConfig

	// Copy Metadata fields from bundle.yaml.
	buildSchema.Metadata = bundleSchema.Metadata

	// Merge Env fields from bundle.yaml and apphosting.yaml together.
	if len(appHostingSchema.Env) > 0 || len(bundleSchema.Env) > 0 {
		buildSchema.Env = mergeEnvironmentVariables(appHostingSchema.Env, bundleSchema.Env)
	}

	return buildSchema
}

// mergeEnvironmentVariables merges the environment variables from apphosting.yaml and bundle.yaml.
// If there is a conflict between the environment variables, use the value/secret from apphosting.yaml.
func mergeEnvironmentVariables(aevs []apphostingschema.EnvironmentVariable, bevs []bundleschema.EnvironmentVariable) []apphostingschema.EnvironmentVariable {
	merged := aevs
	varByName := make(map[string]apphostingschema.EnvironmentVariable)
	for _, apphostingEv := range aevs {
		varByName[apphostingEv.Variable] = apphostingEv
	}

	for _, bundleEv := range bevs {
		apphostingEv, found := varByName[bundleEv.Variable]
		if found && isEnvAvailabilityOverlap(apphostingEv.Availability, bundleEv.Availability) {
			log.Printf("Apphosting.yaml environment variable %v conflicts with bundle.yaml environment variable\n", bundleEv.Variable)
			log.Printf("Using an environment variable value or secret from apphosting.yaml\n")
		} else {
			var ev apphostingschema.EnvironmentVariable = apphostingschema.EnvironmentVariable(bundleEv)
			log.Printf("Adding environment variable %v from bundle.yaml\n", bundleEv.Variable)
			// merge bundleEv in if no conflict
			merged = append(merged, ev)
		}
	}
	return merged
}

func isEnvAvailabilityOverlap(appHostingAvailability, bundleAvailability []string) bool {
	availabilityByName := make(map[string]bool)
	for _, av := range appHostingAvailability {
		availabilityByName[av] = true
	}
	for _, av := range bundleAvailability {
		if availabilityByName[av] {
			return true
		}
	}
	return false
}

// Publish takes in the path to various required files such as apphosting.yaml, bundle.yaml, and
// other files (tbd) and merges them into one output that describes the desired Backend Service
// configuration before pushing this information to the control plane.
func Publish(appHostingYAMLPath string, bundleYAMLPath string, outputFilePath string) error {
	appHostingSchema, err := apphostingschema.ReadAndValidateFromFile(appHostingYAMLPath)
	if err != nil {
		return err
	}

	// For now, simply validates that bundle.yaml exists.
	bundleSchema, err := bundleschema.ReadAndValidateFromFile(bundleYAMLPath)
	if err != nil {
		return err
	}

	buildSchema := toBuildSchema(appHostingSchema, bundleSchema)

	err = writeToFile(buildSchema, outputFilePath)
	if err != nil {
		return err
	}

	return nil
}
