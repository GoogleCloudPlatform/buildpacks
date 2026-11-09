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

package webbundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CheckStandalone verifies that <serverDir>/server.js exists as a regular file
// and node_modules/next/package.json exists in serverDir or a parent folder up
// to the standalone root.
func CheckStandalone(workspaceRoot string, schema *BundleSchema) error {
	if schema == nil {
		return errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}

	serverJSRel := path.Join(schema.Metadata.ServerDir, "server.js")
	if _, err := validateRegularFile(realRoot, serverJSRel); err != nil {
		return fmt.Errorf("checking %s: %v", serverJSRel, err)
	}

	stopDir := standaloneStopDir(schema)
	curDir := path.Clean(schema.Metadata.ServerDir)
	foundNextPkg := false
	for {
		if !isCoveredByInclude(curDir, schema.OutputFiles.ServerApp.Include) {
			break
		}
		nextPkgRel := path.Join(curDir, "node_modules/next/package.json")
		if _, err := validateRegularFile(realRoot, nextPkgRel); err == nil {
			foundNextPkg = true
			break
		}
		if curDir == stopDir || curDir == "." || curDir == "/" {
			break
		}
		parent := path.Dir(curDir)
		if parent == curDir {
			break
		}
		curDir = parent
	}
	if !foundNextPkg {
		return fmt.Errorf("node_modules/next/package.json not found in %s or parent directories up to %s", schema.Metadata.ServerDir, stopDir)
	}

	if UsesServeJS(schema.RunConfig.RunCommand) {
		distDirRel := relativeDistDir(schema.Metadata.ProjectDir, schema.Metadata.DistDir)
		reqFilesRel := path.Join(schema.Metadata.ServerDir, distDirRel, "required-server-files.json")
		if err := validateNonEmptyJSONFile(realRoot, reqFilesRel); err != nil {
			return fmt.Errorf("checking standalone required-server-files.json at %s: %v", reqFilesRel, err)
		}
	}

	return nil
}

func relativeDistDir(projectDir, distDir string) string {
	projectDir = path.Clean(projectDir)
	distDir = path.Clean(distDir)
	if projectDir == "." || projectDir == "" {
		return distDir
	}
	prefix := projectDir + "/"
	if strings.HasPrefix(distDir, prefix) {
		return strings.TrimPrefix(distDir, prefix)
	}
	return distDir
}

func standaloneStopDir(schema *BundleSchema) string {
	standaloneRoot := path.Join(schema.Metadata.DistDir, "standalone")
	if isSubpathOrEqual(schema.Metadata.ServerDir, standaloneRoot) && isCoveredByInclude(standaloneRoot, schema.OutputFiles.ServerApp.Include) {
		return standaloneRoot
	}
	stop := schema.Metadata.ServerDir
	for _, inc := range schema.OutputFiles.ServerApp.Include {
		if isSubpathOrEqual(schema.Metadata.ServerDir, inc) && isSubpathOrEqual(stop, inc) {
			stop = path.Clean(inc)
		}
	}
	return stop
}

// CheckRuntimeManifest verifies that schema.Metadata.RuntimeManifest does not
// escape workspaceRoot and parses as a valid non-empty JSON object. When either
// RunCommand or ProjectRunCommand uses .cloudrun/serve.js, the runtime manifest
// must exist on disk.
func CheckRuntimeManifest(workspaceRoot string, schema *BundleSchema) error {
	if schema == nil {
		return errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}
	manifestRel := schema.Metadata.RuntimeManifest
	if manifestRel == "" {
		manifestRel = DefaultRuntimeManifestRelPath
	}
	if _, err := CleanAndValidatePath(realRoot, manifestRel, false); err != nil {
		return err
	}
	manifestAbs := filepath.Join(realRoot, filepath.FromSlash(manifestRel))
	if _, err := os.Lstat(manifestAbs); os.IsNotExist(err) {
		if UsesServeJS(schema.RunConfig.RunCommand) || UsesServeJS(schema.RunConfig.ProjectRunCommand) {
			return fmt.Errorf("runtime manifest %s is required when using .cloudrun/serve.js: %v", manifestRel, err)
		}
		return nil
	}
	return validateNonEmptyJSONFile(realRoot, manifestRel)
}

// CheckBootPackageJSON verifies that package.json exists as a regular file
// inside workspaceRoot in the boot directory for mode (schema.Metadata.ServerDir
// when mode normalizes to "true", or schema.Metadata.ProjectDir for
// "report-only" and "false").
func CheckBootPackageJSON(workspaceRoot string, schema *BundleSchema, mode string) error {
	if schema == nil {
		return errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}
	bootDir := schema.Metadata.ProjectDir
	if strings.TrimSpace(bootDir) == "" {
		bootDir = "."
	}
	if NormalizePruneMode(mode) == PruneModeTrue {
		bootDir = schema.Metadata.ServerDir
		if strings.TrimSpace(bootDir) == "" {
			return errors.New("metadata.serverDir is empty")
		}
	}
	pkgJSONRel := path.Join(bootDir, "package.json")
	if _, err := validateRegularFile(realRoot, pkgJSONRel); err != nil {
		return fmt.Errorf("checking boot package.json at %s: %v", pkgJSONRel, err)
	}
	return nil
}

// CheckProjectFallbackFiles verifies the files that project mode needs inside
// workspaceRoot: <projectDir>/package.json must be a regular file, and the
// runtime manifest (default .cloudrun/next-routing.json) and
// <distDir>/required-server-files.json must be valid non-empty JSON objects.
// The returned error names every file that failed its check.
func CheckProjectFallbackFiles(workspaceRoot string, schema *BundleSchema) error {
	if schema == nil {
		return errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}
	projectDir := schema.Metadata.ProjectDir
	if strings.TrimSpace(projectDir) == "" {
		projectDir = "."
	}
	pkgJSONRel := path.Join(projectDir, "package.json")
	runtimeManifestRel := schema.Metadata.RuntimeManifest
	if runtimeManifestRel == "" {
		runtimeManifestRel = DefaultRuntimeManifestRelPath
	}
	reqServerFilesRel := path.Join(schema.Metadata.DistDir, "required-server-files.json")

	var errs []error
	if _, err := validateRegularFile(realRoot, pkgJSONRel); err != nil {
		errs = append(errs, fmt.Errorf("checking project package.json at %s: %v", pkgJSONRel, err))
	}
	if err := validateNonEmptyJSONFile(realRoot, runtimeManifestRel); err != nil {
		errs = append(errs, fmt.Errorf("checking runtime manifest at %s: %v", runtimeManifestRel, err))
	}
	if err := validateNonEmptyJSONFile(realRoot, reqServerFilesRel); err != nil {
		errs = append(errs, fmt.Errorf("checking required-server-files.json at %s: %v", reqServerFilesRel, err))
	}
	return errors.Join(errs...)
}

// validateRegularFile checks that relPath stays inside realRoot, is a regular
// file, and does not resolve through symlinks to a path outside realRoot. It
// returns the cleaned relative path.
func validateRegularFile(realRoot, relPath string) (string, error) {
	cleaned, err := CleanAndValidatePath(realRoot, relPath, false)
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(realRoot, filepath.FromSlash(cleaned))
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", cleaned)
	}
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	if !isWithinRealRoot(realRoot, resolved) {
		return "", fmt.Errorf("%s resolves outside workspace root", cleaned)
	}
	return cleaned, nil
}

func validateNonEmptyJSONFile(realRoot, relPath string) error {
	cleaned, err := validateRegularFile(realRoot, relPath)
	if err != nil {
		return err
	}
	absPath := filepath.Join(realRoot, filepath.FromSlash(cleaned))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if len(obj) == 0 {
		return fmt.Errorf("%s is an empty JSON object", cleaned)
	}
	return nil
}
