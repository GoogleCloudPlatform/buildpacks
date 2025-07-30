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

// Package bundleschema provides functionality around parsing and managing bundle.yaml.
package bundleschema

import (
	"errors"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"gopkg.in/yaml.v2"
)

var (
	validAvailabilityValues      = map[string]bool{"RUNTIME": true}
	errMissingAdapterPackageName = errors.New("missing the adapter package name in bundle.yaml metadata")
	errMissingAdapterVersion     = errors.New("missing the adapter version in bundle.yaml metadata")
	errMissingFrameworkName      = errors.New("missing the framework name in bundle.yaml metadata")
	errMissingFrameworkVersion   = errors.New("missing the framework version in bundle.yaml metadata")
)

// EnvironmentVariable is the struct representation of the environment variables from bundle.yaml.
type EnvironmentVariable apphostingschema.EnvironmentVariable

// BundleSchema is the struct representation of bundle.yaml.
type BundleSchema struct {
	RunConfig RunConfig `yaml:"runConfig"`
	Metadata  *Metadata `yaml:"metadata,omitempty"`
}

// RunConfig is the struct representation of the passed cloud run config.
type RunConfig struct {
	EnvironmentVariables []EnvironmentVariable       `yaml:"environmentVariables,omitempty"`
	CPU                  *float32                    `yaml:"cpu"`
	MemoryMiB            *int32                      `yaml:"memoryMiB"`
	Concurrency          *int32                      `yaml:"concurrency"`
	MaxInstances         *int32                      `yaml:"maxInstances"`
	MinInstances         *int32                      `yaml:"minInstances"`
	VpcAccess            *apphostingschema.VpcAccess `yaml:"vpcAccess"`
	CPUAlwaysAllocated   *bool                       `yaml:"cpuAlwaysAllocated"`
}

// Metadata is the struct representation of the metadata from bundle.yaml.
type Metadata struct {
	AdapterPackageName string `yaml:"adapterPackageName"`
	AdapterVersion     string `yaml:"adapterVersion"`
	Framework          string `yaml:"framework"`
	// TODO: b/366036980 retrieve community adapter's framework version from buildpack
	FrameworkVersion string `yaml:"frameworkVersion"`
}

// UnmarshalYAML provides custom validation logic to validate bundle.yaml environment variables.
func (ev *EnvironmentVariable) UnmarshalYAML(unmarshal func(any) error) error {
	type standardYAML EnvironmentVariable // Define an alias
	// Use alias and standard unmarshal to avoid recursive unmarshal on EnvironmentVariable fields
	if err := unmarshal((*standardYAML)(ev)); err != nil {
		return err
	}

	if ev.Value == "" || ev.Secret != "" {
		return fmt.Errorf("for bundle.yaml environment variable %q, 'value' is required and 'secret' should not be present", ev.Variable)
	}

	for _, val := range ev.Availability {
		if !validAvailabilityValues[val] {
			return fmt.Errorf("invalid value %s in 'availability'", val)
		}
	}

	return nil
}

// UnmarshalYAML provides custom validation logic to validate bundle.yaml metadata.
func (md *Metadata) UnmarshalYAML(unmarshal func(any) error) error {
	type standardYAML Metadata // Define an alias
	// Use alias and standard unmarshal to avoid recursive unmarshal on Metadata fields
	if err := unmarshal((*standardYAML)(md)); err != nil {
		return err
	}

	if md.AdapterPackageName == "" {
		return errMissingAdapterPackageName
	}
	if md.AdapterVersion == "" {
		return errMissingAdapterVersion
	}
	if md.Framework == "" {
		return errMissingFrameworkName
	}
	if md.FrameworkVersion == "" {
		return errMissingFrameworkVersion
	}

	return nil
}

// ReadAndValidateFromFile converts the provided file into an BundleSchema.
func ReadAndValidateFromFile(filePath string) (BundleSchema, error) {
	var b BundleSchema
	bundleBuffer, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return b, fmt.Errorf("missing output bundle config at %v", filePath)
	} else if err != nil {
		return b, fmt.Errorf("reading output bundle config at %v: %w", filePath, err)
	}

	if err = yaml.Unmarshal(bundleBuffer, &b); err != nil {
		return b, fmt.Errorf("unmarshalling apphosting config as YAML: %w", err)
	}

	return b, nil
}
