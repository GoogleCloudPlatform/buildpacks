// Copyright 2025 Google LLC
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

// Implements nodejs/firebasebundle buildpack.
// The output bundle buildpack sets up the output bundle for future steps
// It will do the following
// 1. Copy over static assets to the output bundle dir
// 2. Override run script with a new one to run the optimized build
package lib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/fileutil"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"gopkg.in/yaml.v2"
)

const (
	defaultPublicDir        = "public"
	firebaseOutputBundleDir = "FIREBASE_OUTPUT_BUNDLE_DIR"
	// TODO(b/401016345): Remove this env var once there is a better way to pass the apphosting.yaml file path in tests.
	apphostingYamlPathTestsEnv        = "APPHOSTINGYAML_FILEPATH_TESTS"
	environmentName                   = "ENVIRONMENT_NAME"
	apphostingPreprocessedPathForPack = "/workspace/apphosting_preprocessed"
)

// DetectFn detects if it is a firebase apphosting application.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	// This buildpack handles some necessary setup for future app hosting processes,
	// it should always run for any app hosting initial build.
	if !env.IsFAH() {
		return gcp.OptOut("not a firebase apphosting application"), nil
	}
	// The environment variable is converted to a string "true" not true.
	useGeneric, err := env.IsPresentAndTrue(env.GoogleUseGenericFirebaseBundle)
	if err != nil {
		ctx.Warnf("failed to parse %s: %v", env.GoogleUseGenericFirebaseBundle, err)
	}
	if useGeneric {
		return gcp.OptOut("not using google.nodejs.firebasebundle because GOOGLE_USE_GENERIC_FIREBASEBUNDLE is true"), nil
	}
	return gcp.OptIn("firebase apphosting application"), nil
}

// BuildFn sets up the output bundle for future steps.
func BuildFn(ctx *gcp.Context) error {
	bundlePath := filepath.Join(ctx.ApplicationRoot(), ".apphosting", "bundle.yaml")
	bundleYaml, err := readBundleYaml(ctx, bundlePath)
	if err != nil {
		return err
	}

	outputBundleDir, ok := os.LookupEnv(firebaseOutputBundleDir)

	if !ok {
		return gcp.InternalErrorf("looking up output bundle env %s", firebaseOutputBundleDir)
	}

	workspacePublicDir := filepath.Join(ctx.ApplicationRoot(), defaultPublicDir)
	outputPublicDir := filepath.Join(outputBundleDir, defaultPublicDir)
	appDir := util.ApplicationDirectory(ctx)

	apphostingYamlPathTests, ok := os.LookupEnv(apphostingYamlPathTestsEnv)

	var apphostingYaml *apphostingYaml
	if ok {
		apphostingYaml, err = readAppHostingYaml(ctx, apphostingYamlPathTests)
	} else {
		apphostingYaml, err = readAppHostingYaml(ctx, apphostingPreprocessedPathForPack)
	}

	if err != nil {
		return err
	}
	if bundleYaml == nil {
		ctx.Logf("bundle.yaml does not exist, assuming default configs")

		err = generateDefaultBundleYaml(bundlePath, ctx)
		if err != nil {
			return err
		}
	}

	ctx.Logf("Copying static assets.")
	err = fileutil.CopyFile(filepath.Join(outputBundleDir, "bundle.yaml"), bundlePath)
	if err != nil {
		return gcp.InternalErrorf("copying output bundle dir %s: %w", outputBundleDir, err)
	}
	err = copyPublicDirToOutputBundleDir(outputPublicDir, workspacePublicDir, ctx)
	if err != nil {
		return err
	}
	nodeDeps, err := nodejs.ReadNodeDependencies(ctx, appDir)
	// We don't want to fail builds for failing to collect optional metadata. Ignore error.
	if err == nil {
		setMetadata(nodeDeps.PackageJSON)
	}
	err = deleteFilesNotIncluded(apphostingYaml, bundleYaml, ctx.ApplicationRoot())
	if err != nil {
		return err
	}

	err = SetRunCommand(apphostingYaml, bundleYaml, ctx)
	if err != nil {
		return err
	}
	return nil
}

// apphostingYaml represents the relevant contents of a apphosting.yaml file.
type apphostingYaml struct {
	OutputFiles outputFiles `yaml:"outputFiles,omitempty"`
	Scripts     scripts     `yaml:"scripts,omitempty"`
}

// bundleYaml represents the relevant contents of a bundle.yaml file.
type bundleYaml struct {
	Version     string      `yaml:"version"`
	RunConfig   runConfig   `yaml:"runConfig"`
	OutputFiles outputFiles `yaml:"outputFiles,omitempty"`
}

// runConfig is the struct representation of the passed run config.
type runConfig struct {
	RunCommand string `yaml:"runCommand"`
}

// outputFiles is the struct representation of the passed output files.
type outputFiles struct {
	ServerApp serverApp `yaml:"serverApp"`
}

// serverApp is the struct representation of the passed server app files.
type serverApp struct {
	Include []string `yaml:"include"`
}

// scripts is the struct representation of the apphosting.yaml scripts section.
type scripts struct {
	RunCommand string `yaml:"runCommand"`
}

func readBundleYaml(ctx *gcp.Context, bundlePath string) (*bundleYaml, error) {
	bundleYamlExists, err := ctx.FileExists(bundlePath)
	if err != nil {
		return nil, err
	}
	if !bundleYamlExists {
		// return an empty struct if the file doesn't exist
		return nil, nil
	}
	rawBundleYaml, err := ctx.ReadFile(bundlePath)
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %w", bundlePath, err)
	}
	var bundleYaml bundleYaml
	if err := yaml.Unmarshal(rawBundleYaml, &bundleYaml); err != nil {
		return nil, gcp.UserErrorf("invalid %s: %w", bundlePath, err)
	}
	return &bundleYaml, nil
}

func readAppHostingYaml(ctx *gcp.Context, appHostingPath string) (*apphostingYaml, error) {
	appHostingYamlExists, err := ctx.FileExists(appHostingPath)
	if err != nil {
		return nil, err
	}
	if !appHostingYamlExists {
		// return an empty struct if the file doesn't exist
		return nil, nil
	}
	rawAppHostingYaml, err := ctx.ReadFile(appHostingPath)
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %w", appHostingPath, err)
	}
	var appHostingYaml apphostingYaml
	if err := yaml.Unmarshal(rawAppHostingYaml, &appHostingYaml); err != nil {
		return nil, gcp.UserErrorf("invalid %s: %w", appHostingPath, err)
	}
	return &appHostingYaml, nil
}

func generateDefaultBundleYaml(bundleYamlPath string, ctx *gcp.Context) error {
	ctx.MkdirAll(path.Dir(bundleYamlPath), 0744)
	f, err := ctx.CreateFile(filepath.Join(bundleYamlPath))
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func copyPublicDirToOutputBundleDir(outputPublicDir string, workspacePublicDir string, ctx *gcp.Context) error {
	publicDirExists, err := ctx.FileExists(workspacePublicDir)
	if err != nil {
		return err
	}
	if !publicDirExists {
		return nil
	}
	err = ctx.MkdirAll(outputPublicDir, 0744)
	if err != nil {
		return err
	}
	if err := fileutil.MaybeCopyPathContents(outputPublicDir, workspacePublicDir, fileutil.AllPaths); err != nil {
		return err
	}
	return nil
}

// detect if the app uses AI (Genkit or GenAI API) and the corresponding version, and set metadata accordingly
func setMetadata(packageJSON *nodejs.PackageJSON) {
	if packageJSON != nil {
		genkitVersion := nodejs.DependencyVersion(packageJSON, "genkit")
		genAIVersion := nodejs.DependencyVersion(packageJSON, "@google/generative-ai")
		if genkitVersion != "" {
			buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.IsUsingGenkit, buildermetadata.MetadataValue(genkitVersion))
		}
		if genAIVersion != "" {
			buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.IsUsingGenAI, buildermetadata.MetadataValue(genAIVersion))
		}
	}
}

// parseCommand parses a command string into a list of arguments.
// The command string is expected to be in the format of a shell command.
// It will split the command string on spaces and remove any leading and trailing spaces.
// It will also handle quoted strings and arguments.
// For example, the command string "npm run start" will be parsed into the list ["npm", "run", "start"].
// The command string `npm run "node package" test` will be parsed into the list ["npm", "run", "node package", "test"].
// The command string `mycommand "arg with spaces" anotherarg` will be parsed into the list ["mycommand", "arg with spaces", "anotherarg"].
// The command string `noquotes` will be parsed into the list ["noquotes"].
// The command string `npm run "node package test` will return an error because the quotes are not closed.
func parseCommand(command string) ([]string, error) {
	var result []string
	var currCmdUnit strings.Builder
	inQuotes := false

	for _, r := range command {
		if r == '"' {
			inQuotes = !inQuotes
			if !inQuotes {
				result = append(result, currCmdUnit.String())
				currCmdUnit.Reset()
			}
			continue
		}

		if unicode.IsSpace(r) && !inQuotes {
			if currCmdUnit.Len() > 0 {
				result = append(result, currCmdUnit.String())
				currCmdUnit.Reset()
			}
			continue
		}

		currCmdUnit.WriteRune(r)
	}

	if currCmdUnit.Len() > 0 {
		result = append(result, currCmdUnit.String())
	}

	if inQuotes {
		return nil, gcp.UserErrorf("parsing command, there was an unclosed quote %s", command)
	}
	return result, nil
}

// SetRunCommand sets the run command for the web process.
// The run command is set from the apphosting.yaml file if it exists, otherwise it is set from the bundle.yaml file.
// If neither exists, the run command is set to the default "npm run start".
func SetRunCommand(apphostingYaml *apphostingYaml, bundleYaml *bundleYaml, ctx *gcp.Context) error {
	if apphostingYaml != nil && apphostingYaml.Scripts.RunCommand != "" {
		ctx.Logf("Setting run command from apphosting.yaml: %s", apphostingYaml.Scripts.RunCommand)
		parsedCommand, err := parseCommand(apphostingYaml.Scripts.RunCommand)
		if err != nil {
			return err
		}
		ctx.AddWebProcess(parsedCommand)
		return nil
	}
	if bundleYaml != nil && bundleYaml.RunConfig.RunCommand != "" {
		ctx.Logf("Setting run command from bundle.yaml: %s", bundleYaml.RunConfig.RunCommand)
		parsedCommand, err := parseCommand(bundleYaml.RunConfig.RunCommand)
		if err != nil {
			return err
		}
		ctx.AddWebProcess(parsedCommand)
	}
	return nil
}

func convertToMap(slice []string) map[string]bool {
	var newMap map[string]bool
	newMap = make(map[string]bool)
	for _, s := range slice {
		newMap[s] = true
	}
	return newMap
}

func extractAllDirs(files []string) []string {
	var result []string
	for _, file := range files {
		dir := file
		for dir != "." && dir != "/" { // Stop at root directory
			result = append(result, dir)
			dir = filepath.Dir(dir)
		}
	}
	return result
}

// walkDirStructureAndDeleteAllFilesNotIncluded walks the directory structure and deletes all files that are not included.
// If a directory is labeled as included, all files in that directory will be kept.
// "." in either apphosting.yaml or bundle.yaml will include all files.
func walkDirStructureAndDeleteAllFilesNotIncluded(rootDir string, filesToInclude []string, dirsToIncludeAll []string) error {
	filesToIncludeMap := convertToMap(filesToInclude)

	dirsToIncludeAllMap := convertToMap(dirsToIncludeAll)

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // Skip this file/directory
			}

			return fmt.Errorf("walking directory structure: %w", err)
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		// Keep the root directory
		if relPath == "." {
			return nil
		}

		if !filesToIncludeMap[relPath] && !anyParentDirMatches(relPath, dirsToIncludeAllMap) {
			if info.IsDir() {
				return os.RemoveAll(path)
			}
			return os.Remove(path)
		}

		return nil
	})
}

func anyParentDirMatches(path string, targets map[string]bool) bool {
	dir := path
	for dir != "." && dir != "/" { // Stop at the root directory
		if targets[dir] {
			return true
		}

		dir = filepath.Dir(dir) // Move to the parent directory
	}

	return false
}

// deleteFilesNotIncluded deletes all files that are not included in the apphosting.yaml or bundle.yaml.
// This is done by walking the directory structure and deleting all files that are not included.
// if a directory is labeled as included, all files in that directory will be kept.
// "." in either apphosting.yaml or bundle.yaml will include all files.
func deleteFilesNotIncluded(apphostingSchema *apphostingYaml, bundleSchema *bundleYaml, appPath string) error {
	// always include apphosting.yaml
	includedFiles := originalApphostingYamlPaths()
	// always include all of .apphosting
	fullyIncludedDirs := []string{".apphosting"}
	if bundleSchema != nil && bundleSchema.OutputFiles.ServerApp.Include != nil {
		includedFiles = append(extractAllDirs(bundleSchema.OutputFiles.ServerApp.Include), includedFiles...)
		fullyIncludedDirs = append(bundleSchema.OutputFiles.ServerApp.Include, fullyIncludedDirs...)
	}
	if apphostingSchema != nil && apphostingSchema.OutputFiles.ServerApp.Include != nil {
		includedFiles = append(extractAllDirs(apphostingSchema.OutputFiles.ServerApp.Include), includedFiles...)
		fullyIncludedDirs = append(apphostingSchema.OutputFiles.ServerApp.Include, fullyIncludedDirs...)
	}
	// if both apphosting.yaml and bundle.yaml are empty, don't delete anything
	if (apphostingSchema == nil || apphostingSchema.OutputFiles.ServerApp.Include == nil) && (bundleSchema == nil || bundleSchema.OutputFiles.ServerApp.Include == nil) {
		return nil
	}
	// Check if "." is present in either include list
	for _, dir := range fullyIncludedDirs {
		if dir == "." {
			// If "." is present, don't delete anything
			return nil
		}
	}

	return walkDirStructureAndDeleteAllFilesNotIncluded(appPath, includedFiles, fullyIncludedDirs)
}

func originalApphostingYamlPaths() []string {
	paths := []string{"apphosting.yaml"}
	envName, ok := os.LookupEnv(environmentName)
	if ok {
		paths = append(paths, fmt.Sprintf("apphosting.%v.yaml", envName))
	}
	return paths
}
