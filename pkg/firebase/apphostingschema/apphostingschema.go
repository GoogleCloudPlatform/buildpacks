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

// Package apphostingschema provides functionality around parsing and managing apphosting.yaml.
package apphostingschema

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/faherror"
	"gopkg.in/yaml.v2"
)

var (
	validAvailabilityValues = map[string]bool{"BUILD": true, "RUNTIME": true}
	reservedKeys            = map[string]bool{
		"PORT":            true,
		"K_SERVICE":       true,
		"K_REVISION":      true,
		"K_CONFIGURATION": true,
	}

	reservedFirebaseKeyPrefix = "X_FIREBASE_"
)

// AppHostingSchema is the struct representation of apphosting.yaml.
type AppHostingSchema struct {
	RunConfig   RunConfig             `yaml:"runConfig,omitempty"`
	Env         []EnvironmentVariable `yaml:"env,omitempty"`
	Scripts     Scripts               `yaml:"scripts,omitempty"`
	OutputFiles OutputFiles           `yaml:"outputFiles,omitempty"`
}

// NetworkInterface is the struct representation of the passed network interface in VPC direct connect.
type NetworkInterface struct {
	Network    string   `yaml:"network,omitempty"`
	Subnetwork string   `yaml:"subnetwork,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

// VpcAccess is the struct representation of the passed vpc access.
type VpcAccess struct {
	Connector         string             `yaml:"connector,omitempty"`
	Egress            string             `yaml:"egress,omitempty"`
	NetworkInterfaces []NetworkInterface `yaml:"networkInterfaces,omitempty"`
}

// RunConfig is the struct representation of the passed run config.
type RunConfig struct {
	// value types used must match server field types. pointers are used to capture unset vs zero-like values.
	CPU                *float32   `yaml:"cpu"`
	MemoryMiB          *int32     `yaml:"memoryMiB"`
	Concurrency        *int32     `yaml:"concurrency"`
	MaxInstances       *int32     `yaml:"maxInstances"`
	MinInstances       *int32     `yaml:"minInstances"`
	VpcAccess          *VpcAccess `yaml:"vpcAccess"`
	CPUAlwaysAllocated *bool      `yaml:"cpuAlwaysAllocated"`
}

// EnvironmentVariable is the struct representation of the passed environment variables.
type EnvironmentVariable struct {
	Variable     string   `yaml:"variable"`
	Value        string   `yaml:"value,omitempty"`  // Optional: Can be value xor secret
	Secret       string   `yaml:"secret,omitempty"` // Optional: Can be value xor secret
	Availability []string `yaml:"availability,omitempty"`
}

// Scripts is the struct representation of the scripts in apphosting.yaml.
type Scripts struct {
	RunCommand   string `yaml:"runCommand,omitempty"`
	BuildCommand string `yaml:"buildCommand,omitempty"`
}

// OutputFiles is the struct representation of the passed output files.
type OutputFiles struct {
	ServerApp serverApp `yaml:"serverApp"`
}

// serverApp is the struct representation of the passed server app files.
type serverApp struct {
	Include []string `yaml:"include"`
}

// UnmarshalYAML provides custom validation logic to validate EnvironmentVariable
func (ev *EnvironmentVariable) UnmarshalYAML(unmarshal func(any) error) error {
	type plain EnvironmentVariable // Define an alias
	if err := unmarshal((*plain)(ev)); err != nil {
		return err
	}

	if ev.Value != "" && ev.Secret != "" {
		return fmt.Errorf("both 'value' and 'secret' fields cannot be present")
	}

	if ev.Value == "" && ev.Secret == "" {
		return fmt.Errorf("either 'value' or 'secret' field is required")
	}

	for _, val := range ev.Availability {
		if !validAvailabilityValues[val] {
			return fmt.Errorf("invalid value in 'availability': %s", val)
		}
	}

	return nil
}

// UnmarshalYAML provides custom validation logic to validate RunConfig
func (rc *RunConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type plain RunConfig // Define an alias
	if err := unmarshal((*plain)(rc)); err != nil {
		return err
	}

	// Validation for 'CPU'
	if rc.CPU != nil && !(1 <= *rc.CPU && *rc.CPU <= 8) {
		return fmt.Errorf("runConfig.cpu field is not in valid range of [1, 8]")
	}

	// Validation for 'MemoryMiB'
	if rc.MemoryMiB != nil && !(512 <= *rc.MemoryMiB && *rc.MemoryMiB <= 32768) {
		return fmt.Errorf("runConfig.memory field is not in valid range of [512, 32768]")
	}

	// Validation for 'Concurrency'
	if rc.Concurrency != nil && !(1 <= *rc.Concurrency && *rc.Concurrency <= 1000) {
		return fmt.Errorf("runConfig.concurrency field is not in valid range of [1, 1000]")
	}

	// Validation for 'MaxInstances'
	if rc.MaxInstances != nil && !(1 <= *rc.MaxInstances && *rc.MaxInstances <= 100) {
		return fmt.Errorf("runConfig.maxInstances field is not in valid range of [1, 100]")
	}

	// Validation for 'minInstances'
	if rc.MinInstances != nil && !(0 <= *rc.MinInstances && *rc.MinInstances <= 100) {
		return fmt.Errorf("runConfig.minInstances field is not in valid range of [1, 100]")
	}

	if err := ValidateVpcAccess(rc.VpcAccess); err != nil {
		return err
	}
	return nil
}

// ReadAndValidateFromFile converts the provided file into an AppHostingSchema.
// Returns an empty AppHostingSchema{} if the file does not exist.
func ReadAndValidateFromFile(filePath string) (AppHostingSchema, error) {
	var a AppHostingSchema
	apphostingBuffer, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return a, nil
	} else if err != nil {
		return a, fmt.Errorf("reading apphosting config at %v: %w", filePath, err)
	}

	if err = yaml.Unmarshal(apphostingBuffer, &a); err != nil {
		return a, faherror.InvalidAppHostingYamlError(filePath, err)
	}
	return a, nil
}

func isReservedKey(envKey string) bool {
	if _, ok := reservedKeys[envKey]; ok {
		return true
	} else if strings.HasPrefix(envKey, reservedFirebaseKeyPrefix) {
		return true
	}
	return false
}

func santizeEnv(env []EnvironmentVariable) []EnvironmentVariable {
	if env == nil {
		return nil
	}

	var sanitizedSchemaEnv []EnvironmentVariable
	for _, ev := range env {
		if !isReservedKey(ev.Variable) {
			if ev.Availability == nil {
				log.Printf("%s has no availability specified, applying the default of 'BUILD' and 'RUNTIME'", ev.Variable)
				ev.Availability = []string{"BUILD", "RUNTIME"}
			}
			sanitizedSchemaEnv = append(sanitizedSchemaEnv, ev)
		} else {
			log.Printf("WARNING: %s is a reserved key, removing it from the final environment variables", ev.Variable)
		}
	}

	return sanitizedSchemaEnv
}

// Sanitize strips reserved environment variables from the environment variable
// list.
func Sanitize(schema *AppHostingSchema) {
	schema.Env = santizeEnv(schema.Env)
}

// Merge app hosting schemas with priority given to any environment specific overrides
func mergeAppHostingSchemas(appHostingSchema *AppHostingSchema, envSpecificSchema *AppHostingSchema) {
	// Merge RunConfig
	if envSpecificSchema.RunConfig.CPU != nil {
		appHostingSchema.RunConfig.CPU = envSpecificSchema.RunConfig.CPU
	}
	if envSpecificSchema.RunConfig.MemoryMiB != nil {
		appHostingSchema.RunConfig.MemoryMiB = envSpecificSchema.RunConfig.MemoryMiB
	}
	if envSpecificSchema.RunConfig.Concurrency != nil {
		appHostingSchema.RunConfig.Concurrency = envSpecificSchema.RunConfig.Concurrency
	}
	if envSpecificSchema.RunConfig.MaxInstances != nil {
		appHostingSchema.RunConfig.MaxInstances = envSpecificSchema.RunConfig.MaxInstances
	}
	if envSpecificSchema.RunConfig.MinInstances != nil {
		appHostingSchema.RunConfig.MinInstances = envSpecificSchema.RunConfig.MinInstances
	}
	if envSpecificSchema.Scripts.BuildCommand != "" {
		appHostingSchema.Scripts.BuildCommand = envSpecificSchema.Scripts.BuildCommand
	}
	if envSpecificSchema.Scripts.RunCommand != "" {
		appHostingSchema.Scripts.RunCommand = envSpecificSchema.Scripts.RunCommand
	}
	appHostingSchema.RunConfig.VpcAccess = MergeVpcAccess(appHostingSchema.RunConfig.VpcAccess, envSpecificSchema.RunConfig.VpcAccess)

	// Merge Environment Variables
	appHostingSchema.Env = MergeEnvVars(appHostingSchema.Env, envSpecificSchema.Env)

	// Merge OutputFiles
	// If the override schema includes any files, it replaces the entire list from the base schema.
	if len(envSpecificSchema.OutputFiles.ServerApp.Include) > 0 {
		appHostingSchema.OutputFiles.ServerApp.Include = envSpecificSchema.OutputFiles.ServerApp.Include
	}
}

// MergeEnvVars merges the environment variables from the original list with the override list.
// If there is a conflict between the environment variables, use the value/secret from the override list.
func MergeEnvVars(original, override []EnvironmentVariable) []EnvironmentVariable {
	merged := override
	varByName := make(map[string]EnvironmentVariable)
	for _, ev := range override {
		varByName[ev.Variable] = ev
	}

	for _, ev := range original {
		if _, found := varByName[ev.Variable]; !found {
			merged = append(merged, ev)
		} else {
			log.Printf("Skipping environment variable %v from original list since it is already defined in the override list\n", ev.Variable)
		}
	}

	return merged
}

// MergeWithEnvironmentSpecificYAML merges the environment specific apphosting.<environmentName>.yaml with the base apphosting schema found in apphosting.yaml
func MergeWithEnvironmentSpecificYAML(appHostingSchema *AppHostingSchema, appHostingYAMLPath string, environmentName string) error {
	if environmentName == "" {
		return nil
	}

	envSpecificYAMLPath := filepath.Join(filepath.Dir(appHostingYAMLPath), fmt.Sprintf("apphosting.%v.yaml", environmentName))
	envSpecificSchema, err := ReadAndValidateFromFile(envSpecificYAMLPath)
	if err != nil {
		return fmt.Errorf("reading environment specific apphosting schema: %w", err)
	}

	mergeAppHostingSchemas(appHostingSchema, &envSpecificSchema)
	return nil
}

// IsKeyUserDefined determines whether the provided KEY environment variable is already user defined.
func IsKeyUserDefined(schema *AppHostingSchema, key string) bool {
	for _, e := range schema.Env {
		if e.Variable == key {
			return true
		}
	}
	return false
}

// WriteToFile writes the given app hosting schema to the specified path.
func (schema *AppHostingSchema) WriteToFile(outputFilePath string) error {
	fileData, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("converting struct to YAML: %w", err)
	}
	log.Printf("Final app hosting schema:\n%v\n", string(fileData))

	if err := os.MkdirAll(filepath.Dir(outputFilePath), os.ModeDir); err != nil {
		return fmt.Errorf("creating parent directory %q: %w", outputFilePath, err)
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("creating app hosting schema file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(fileData); err != nil {
		return fmt.Errorf("writing app hosting schema data to file: %w", err)
	}

	return nil
}
