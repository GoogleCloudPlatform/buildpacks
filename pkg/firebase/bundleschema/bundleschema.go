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
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"gopkg.in/yaml.v2"
)

var (
	validAvailabilityValues = map[string]bool{"RUNTIME": true}
)

// EnvironmentVariable is the struct representation of the environment variables from bundle.yaml
type EnvironmentVariable apphostingschema.EnvironmentVariable

// BundleSchema is the struct representation of bundle.yaml.
type BundleSchema struct {
	Env []EnvironmentVariable `yaml:"env,omitempty"`
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
