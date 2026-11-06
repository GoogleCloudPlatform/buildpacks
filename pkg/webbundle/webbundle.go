// Copyright 2026 Google LLC
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

// Package webbundle parses, validates, copies, and prunes .cloudrun/bundle.yaml
// v2 manifests.
package webbundle

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
)

const (
	// EnvWebPrune is the environment variable controlling image pruning mode.
	EnvWebPrune = "GOOGLE_WEB_PRUNE"

	// SchemaVersionV2 is the supported schema version for .cloudrun/bundle.yaml.
	SchemaVersionV2 = "v2"

	// OutputModeStandalone indicates a standalone server build.
	OutputModeStandalone = "standalone"
	// OutputModeExport indicates a static export build without a server.
	OutputModeExport = "export"

	// PruneModeTrue deletes all files outside the keep-set.
	PruneModeTrue = "true"
	// PruneModeReportOnly logs what pruning would remove and deletes
	// <distDir>/standalone.
	PruneModeReportOnly = "report-only"
	// PruneModeFalse skips keep-set calculation and deletes <distDir>/standalone.
	PruneModeFalse = "false"

	// CloudRunDirName is the workspace directory where adapter artifacts live.
	CloudRunDirName = ".cloudrun"
	// DefaultManifestRelPath is the workspace-relative path to bundle.yaml.
	DefaultManifestRelPath = ".cloudrun/bundle.yaml"
	// DefaultRuntimeManifestRelPath is the default workspace-relative path to
	// next-routing.json.
	DefaultRuntimeManifestRelPath = ".cloudrun/next-routing.json"
)

var (
	// ErrExportMode is returned when bundle.yaml declares outputMode: export.
	ErrExportMode = errors.New("static export mode does not produce a standalone server")
	// ErrEmptyInclude is returned when outputFiles.serverApp.include is empty.
	ErrEmptyInclude = errors.New("outputFiles.serverApp.include is empty")
)

// BundleSchema represents the contents of .cloudrun/bundle.yaml v2.
type BundleSchema struct {
	Version     string      `yaml:"version"`
	RunConfig   RunConfig   `yaml:"runConfig"`
	Metadata    Metadata    `yaml:"metadata"`
	OutputFiles OutputFiles `yaml:"outputFiles"`
}

// RunConfig holds start commands for pruned and unpruned serving modes.
type RunConfig struct {
	RunCommand        string `yaml:"runCommand"`
	ProjectRunCommand string `yaml:"projectRunCommand,omitempty"`
}

// Metadata holds build and directory layout metadata for the bundle.
type Metadata struct {
	AdapterPackageName string `yaml:"adapterPackageName"`
	AdapterVersion     string `yaml:"adapterVersion"`
	Framework          string `yaml:"framework"`
	FrameworkVersion   string `yaml:"frameworkVersion"`
	BuildID            string `yaml:"buildId,omitempty"`
	OutputMode         string `yaml:"outputMode"`
	ProjectDir         string `yaml:"projectDir"`
	DistDir            string `yaml:"distDir"`
	ServerDir          string `yaml:"serverDir"`
	RuntimeManifest    string `yaml:"runtimeManifest"`
}

// OutputFiles specifies files required at runtime.
type OutputFiles struct {
	ServerApp ServerApp `yaml:"serverApp"`
}

// ServerApp specifies the keep-set roots and pre-prune copy pairs for the
// server app.
type ServerApp struct {
	Include []string   `yaml:"include"`
	Copy    []CopySpec `yaml:"copy,omitempty"`
}

// CopySpec defines a source and destination workspace-relative path pair to
// merge before pruning.
type CopySpec struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// ReadAndValidate reads and validates <workspaceRoot>/.cloudrun/bundle.yaml.
// It returns ErrExportMode when outputMode is "export" and ErrEmptyInclude when
// outputFiles.serverApp.include is empty.
func ReadAndValidate(workspaceRoot string) (*BundleSchema, error) {
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	if _, err := CleanAndValidatePath(realRoot, DefaultManifestRelPath, false); err != nil {
		return nil, fmt.Errorf("validating manifest path %q: %v", DefaultManifestRelPath, err)
	}
	manifestPath := filepath.Join(realRoot, filepath.FromSlash(DefaultManifestRelPath))
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %v", DefaultManifestRelPath, err)
	}
	return ParseAndValidate(realRoot, data)
}

// ParseAndValidate unmarshals raw YAML bytes and validates the resulting
// BundleSchema. It returns ErrExportMode when outputMode is "export" and
// ErrEmptyInclude when outputFiles.serverApp.include is empty.
func ParseAndValidate(workspaceRoot string, data []byte) (*BundleSchema, error) {
	var schema BundleSchema
	if err := nodejs.DecodeYAML(data, &schema); err != nil {
		return nil, fmt.Errorf("unmarshaling bundle.yaml: %v", err)
	}
	if err := ValidateAndNormalize(workspaceRoot, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ValidateAndNormalize validates and normalizes all fields and paths in schema.
// It returns ErrExportMode when outputMode is "export" and ErrEmptyInclude when
// outputFiles.serverApp.include is empty.
func ValidateAndNormalize(workspaceRoot string, schema *BundleSchema) error {
	if schema == nil {
		return errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}

	if schema.Version != SchemaVersionV2 {
		return fmt.Errorf("unsupported bundle.yaml version %q, want %q", schema.Version, SchemaVersionV2)
	}

	mode := strings.TrimSpace(schema.Metadata.OutputMode)
	if mode == "" {
		mode = OutputModeStandalone
	}
	switch mode {
	case OutputModeExport:
		schema.Metadata.OutputMode = OutputModeExport
		return ErrExportMode
	case OutputModeStandalone:
		schema.Metadata.OutputMode = OutputModeStandalone
	default:
		return fmt.Errorf("unsupported metadata.outputMode %q", schema.Metadata.OutputMode)
	}

	schema.RunConfig.RunCommand = strings.TrimSpace(schema.RunConfig.RunCommand)
	schema.RunConfig.ProjectRunCommand = strings.TrimSpace(schema.RunConfig.ProjectRunCommand)
	if schema.RunConfig.RunCommand == "" {
		return errors.New("runConfig.runCommand is empty")
	}

	rawProjectDir := strings.TrimSpace(schema.Metadata.ProjectDir)
	if rawProjectDir == "" {
		rawProjectDir = "."
	}
	projectDir, err := CleanAndValidatePath(realRoot, rawProjectDir, true)
	if err != nil {
		return fmt.Errorf("invalid metadata.projectDir %q: %v", schema.Metadata.ProjectDir, err)
	}
	schema.Metadata.ProjectDir = projectDir

	rawDistDir := strings.TrimSpace(schema.Metadata.DistDir)
	if rawDistDir == "" {
		rawDistDir = path.Join(projectDir, ".next")
	}
	distDir, err := CleanAndValidatePath(realRoot, rawDistDir, false)
	if err != nil {
		return fmt.Errorf("invalid metadata.distDir %q: %v", schema.Metadata.DistDir, err)
	}
	schema.Metadata.DistDir = distDir

	rawServerDir := strings.TrimSpace(schema.Metadata.ServerDir)
	if rawServerDir == "" {
		return errors.New("metadata.serverDir is empty")
	}
	serverDir, err := CleanAndValidatePath(realRoot, rawServerDir, false)
	if err != nil {
		return fmt.Errorf("invalid metadata.serverDir %q: %v", schema.Metadata.ServerDir, err)
	}
	schema.Metadata.ServerDir = serverDir

	if projectDir != "." && !isSubpathOrEqual(distDir, projectDir) {
		prefixedDistDir := path.Join(projectDir, distDir)
		if isSubpathOrEqual(serverDir, prefixedDistDir) {
			if validatedDistDir, distErr := CleanAndValidatePath(realRoot, prefixedDistDir, false); distErr == nil {
				distDir = validatedDistDir
				schema.Metadata.DistDir = distDir
			}
		}
	}

	rawRuntimeManifest := strings.TrimSpace(schema.Metadata.RuntimeManifest)
	if rawRuntimeManifest == "" {
		rawRuntimeManifest = DefaultRuntimeManifestRelPath
	}
	runtimeManifest, err := CleanAndValidatePath(realRoot, rawRuntimeManifest, false)
	if err != nil {
		return fmt.Errorf("invalid metadata.runtimeManifest %q: %v", schema.Metadata.RuntimeManifest, err)
	}
	schema.Metadata.RuntimeManifest = runtimeManifest

	if len(schema.OutputFiles.ServerApp.Include) == 0 {
		return ErrEmptyInclude
	}

	cacheDir := path.Join(distDir, "cache")
	cleanedInclude := make([]string, 0, len(schema.OutputFiles.ServerApp.Include)+1)
	seenInclude := make(map[string]bool, len(schema.OutputFiles.ServerApp.Include)+1)
	for _, rawInc := range schema.OutputFiles.ServerApp.Include {
		inc, err := CleanAndValidatePath(realRoot, rawInc, true)
		if err != nil {
			return fmt.Errorf("invalid outputFiles.serverApp.include path %q: %v", rawInc, err)
		}
		if inc != "." && isSubpathOrEqual(inc, cacheDir) {
			return fmt.Errorf("outputFiles.serverApp.include path %q must not be under %q", rawInc, cacheDir)
		}
		if !seenInclude[inc] {
			seenInclude[inc] = true
			cleanedInclude = append(cleanedInclude, inc)
		}
	}
	if !seenInclude[CloudRunDirName] && !seenInclude["."] {
		seenInclude[CloudRunDirName] = true
		cleanedInclude = append(cleanedInclude, CloudRunDirName)
	}
	schema.OutputFiles.ServerApp.Include = cleanedInclude

	if !isCoveredByInclude(serverDir, cleanedInclude) {
		return fmt.Errorf("metadata.serverDir %q is not under any outputFiles.serverApp.include path", serverDir)
	}

	cleanedCopy := make([]CopySpec, 0, len(schema.OutputFiles.ServerApp.Copy))
	for _, rawCopy := range schema.OutputFiles.ServerApp.Copy {
		fromClean, err := CleanAndValidatePath(realRoot, rawCopy.From, false)
		if err != nil {
			return fmt.Errorf("invalid outputFiles.serverApp.copy.from path %q: %v", rawCopy.From, err)
		}
		toClean, err := CleanAndValidatePath(realRoot, rawCopy.To, false)
		if err != nil {
			return fmt.Errorf("invalid outputFiles.serverApp.copy.to path %q: %v", rawCopy.To, err)
		}
		if !isCoveredByInclude(toClean, cleanedInclude) {
			return fmt.Errorf("outputFiles.serverApp.copy.to %q is not under any outputFiles.serverApp.include path", toClean)
		}
		if isSubpathOrEqual(toClean, fromClean) || isSubpathOrEqual(fromClean, toClean) {
			return fmt.Errorf("outputFiles.serverApp.copy paths %q and %q must not overlap", fromClean, toClean)
		}
		cleanedCopy = append(cleanedCopy, CopySpec{
			From: fromClean,
			To:   toClean,
		})
	}
	schema.OutputFiles.ServerApp.Copy = cleanedCopy

	return nil
}

// IsValidPruneMode reports whether raw is empty or a recognized
// GOOGLE_WEB_PRUNE value.
func IsValidPruneMode(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "true", "1", "false", "0", PruneModeReportOnly:
		return true
	default:
		return false
	}
}

// NormalizePruneMode normalizes a raw GOOGLE_WEB_PRUNE value to "true",
// "report-only", or "false". Empty or unrecognized values default to
// "report-only".
func NormalizePruneMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1":
		return PruneModeTrue
	case "false", "0":
		return PruneModeFalse
	case PruneModeReportOnly, "":
		return PruneModeReportOnly
	default:
		return PruneModeReportOnly
	}
}
