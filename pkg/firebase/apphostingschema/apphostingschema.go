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

const (
	// SourceFirebaseConsole is the source name for environment variables set via the Firebase Console.
	// Think server-side environment variables set via the Firebase Console UI.
	SourceFirebaseConsole = "Firebase Console"

	// SourceFirebaseSystem is the source name for default environment variables provided by Firebase.
	// Think FIREBASE_CONFIG and FIREBASE_WEBAPP_CONFIG.
	SourceFirebaseSystem = "Firebase System"

	// SourceAppHostingYAML is the source name for environment variables defined in apphosting.yaml.
	SourceAppHostingYAML = "apphosting.yaml"

	reservedFirebaseKeyPrefix = "X_FIREBASE_"
)

var (
	validAvailabilityValues = map[string]bool{"BUILD": true, "RUNTIME": true}
	reservedKeys            = map[string]bool{
		"PORT":            true,
		"K_SERVICE":       true,
		"K_REVISION":      true,
		"K_CONFIGURATION": true,
	}
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
	Source       string   `yaml:"source,omitempty"`
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
	// LINT.IfChange(runconfig_validation)
	// --- CPU Validation ---
	// See https://cloud.google.com/run/docs/configuring/cpu for more details.
	if rc.CPU != nil {
		cpu := *rc.CPU
		if !(cpu == 0 || (cpu >= 0.08 && cpu < 1) || cpu == 1 || cpu == 2 || cpu == 4 || cpu == 6 || cpu == 8) {
			return fmt.Errorf("runConfig.cpu: invalid value %f. Valid values are 0, 0.08 to <1, 1, 2, 4, 6, or 8", cpu)
		}
		// Interdependency: CPU vs Concurrency
		if cpu > 0 && cpu < 1 {
			if rc.Concurrency == nil || *rc.Concurrency != 1 {
				// If CPU is fractional, Concurrency must be 1. If Concurrency is not set, it defaults to 80, which is invalid.
				return fmt.Errorf("runConfig.cpu: maximum concurrency must be set to 1 if CPU is less than 1")
			}
		}
	}

	// --- MemoryMiB Validation ---
	if rc.MemoryMiB != nil {
		memoryMiB := *rc.MemoryMiB
		if !(memoryMiB == 0 || (memoryMiB >= 128 && memoryMiB <= 32768)) {
			return fmt.Errorf("runConfig.memoryMiB: invalid value %d. Must be 0 or between 128 and 32768", memoryMiB)
		}
	}

	// --- Concurrency Validation ---
	if rc.Concurrency != nil {
		concurrency := *rc.Concurrency
		if !(concurrency >= 0 && concurrency <= 1000) {
			return fmt.Errorf("runConfig.concurrency: invalid value %d. Must be between 0 and 1000", concurrency)
		}
	}

	// --- Interdependent CPU and MemoryMiB Validation ---
	if rc.CPU != nil && rc.MemoryMiB != nil {
		cpu := *rc.CPU
		memoryMiB := *rc.MemoryMiB
		// Check CPU requirements for Memory
		if memoryMiB >= 24576 && cpu < 8 {
			return fmt.Errorf("runConfig: A minimum of 8 CPUs is required for memory >= 24576 MiB (24 GiB), got CPU %f", cpu)
		}
		if memoryMiB >= 16384 && cpu < 6 {
			return fmt.Errorf("runConfig: A minimum of 6 CPUs is required for memory >= 16384 MiB (16 GiB), got CPU %f", cpu)
		}
		if memoryMiB >= 8192 && cpu < 4 {
			return fmt.Errorf("runConfig: A minimum of 4 CPUs is required for memory >= 8192 MiB (8 GiB), got CPU %f", cpu)
		}
		if memoryMiB >= 4096 && cpu < 2 {
			return fmt.Errorf("runConfig: A minimum of 2 CPUs is required for memory >= 4096 MiB (4 GiB), got CPU %f", cpu)
		}
		if memoryMiB > 1024 && cpu < 1 {
			return fmt.Errorf("runConfig: A minimum of 1 CPU is required for memory > 1024 MiB (1 GiB), got CPU %f", cpu)
		}
		if memoryMiB > 512 && cpu < 0.5 {
			return fmt.Errorf("runConfig: A minimum of 0.5 CPU is required for memory > 512 MiB, got CPU %f", cpu)
		}

		if cpu >= 6 && memoryMiB < 4096 {
			return fmt.Errorf("runConfig: A minimum of 4096 MiB memory is required for 6 CPUs, got %d MiB", memoryMiB)
		}
		if cpu >= 4 && memoryMiB < 2048 {
			return fmt.Errorf("runConfig: A minimum of 2048 MiB memory is required for 4 CPUs, got %d MiB", memoryMiB)
		}
	} else if rc.CPU != nil {
		cpu := *rc.CPU
		if cpu >= 6 {
			return fmt.Errorf("runConfig: A minimum of 4096 MiB memory is required for %f CPUs, but memoryMiB is not set", cpu)
		}
		if cpu >= 4 {
			// Implies default memory (512) is too low
			return fmt.Errorf("runConfig: A minimum of 2048 MiB memory is required for %f CPUs, but memoryMiB is not set", cpu)
		}
	} else if rc.MemoryMiB != nil {
		memoryMiB := *rc.MemoryMiB
		// Implies default CPU of 1
		if memoryMiB >= 4096 {
			return fmt.Errorf("runConfig: A minimum of 2 CPUs is required for memory %d MiB, but CPU is not set", memoryMiB)
		}
	}

	// --- MinInstances Validation ---
	if rc.MinInstances != nil {
		minInst := *rc.MinInstances
		if minInst < 0 {
			return fmt.Errorf("runConfig.minInstances: invalid value %d. Must be >= 0", minInst)
		}
	}

	// --- MaxInstances Validation ---
	if rc.MaxInstances != nil {
		maxInst := *rc.MaxInstances
		if maxInst < 0 {
			return fmt.Errorf("runConfig.maxInstances: invalid value %d. Must be >= 0", maxInst)
		}
		// Interdependency: MaxInstances vs MinInstances
		if rc.MinInstances != nil && maxInst > 0 && *rc.MinInstances > maxInst {
			return fmt.Errorf("runConfig.minInstances (%d) cannot be greater than runConfig.maxInstances (%d)", *rc.MinInstances, maxInst)
		}
	} else if rc.MinInstances != nil && *rc.MinInstances > 100 {
		return fmt.Errorf("runConfig.minInstances: invalid value %d. Must be <= 100 if maxInstances is not set", *rc.MinInstances)
	}
	// LINT.ThenChange(//depot/google3/google/firebase/apphosting/v1main/v1main.proto:runconfig_validation)
	// --- VpcAccess Validation ---
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
	}
	if err != nil {
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
	if envSpecificSchema.RunConfig.CPUAlwaysAllocated != nil {
		appHostingSchema.RunConfig.CPUAlwaysAllocated = envSpecificSchema.RunConfig.CPUAlwaysAllocated
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

	envSpecificYAMLFile := fmt.Sprintf("apphosting.%v.yaml", environmentName)
	envSpecificYAMLPath := filepath.Join(filepath.Dir(appHostingYAMLPath), envSpecificYAMLFile)
	envSpecificSchema, err := ReadAndValidateFromFile(envSpecificYAMLPath)
	if err != nil {
		return fmt.Errorf("reading environment specific apphosting schema: %w", err)
	}
	for i := range envSpecificSchema.Env {
		envSpecificSchema.Env[i].Source = envSpecificYAMLFile
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
