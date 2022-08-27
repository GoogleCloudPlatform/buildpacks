// Copyright 2020 Google LLC
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

// Package dotnet contains .NET buildpack library code.
package dotnet

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	aspDotnetCore        = "Microsoft.AspNetCore.App"
	envSdkVersion        = "GOOGLE_DOTNET_SDK_VERSION"
	envAspNetCoreVersion = "GOOGLE_ASP_NET_CORE_VERSION"
)

// ProjectFiles finds all project files supported by dotnet.
func ProjectFiles(ctx *gcp.Context, dir string) ([]string, error) {
	result, err := ctx.Exec([]string{"find", dir, "-regex", `.*\.\(cs\|fs\|vb\)proj`}, gcp.WithUserTimingAttribution)
	if err != nil {
		return nil, err
	}
	stdout := strings.TrimSpace(result.Stdout)
	if stdout == "" {
		return nil, nil
	}
	return strings.Split(stdout, "\n"), nil
}

// Project represents a .NET project file.
type Project struct {
	XMLName        xml.Name        `xml:"Project"`
	PropertyGroups []PropertyGroup `xml:"PropertyGroup"`
	ItemGroups     []ItemGroup     `xml:"ItemGroup"`
}

// PropertyGroup contains information about a project build.
type PropertyGroup struct {
	AssemblyName     string `xml:"AssemblyName"`
	TargetFramework  string `xml:"TargetFramework"`
	TargetFrameworks string `xml:"TargetFrameworks"`
}

// ItemGroup contains information about a project item group.
type ItemGroup struct {
	PackageReferences []PackageReference `xml:"PackageReference"`
}

// PackageReference contains information about a package reference.
type PackageReference struct {
	Include string `xml:"Include,attr"`
	Version string `xml:"Version,attr"`
}

// ReadProjectFile returns a .NET Project object.
func ReadProjectFile(ctx *gcp.Context, proj string) (Project, error) {
	data, err := ctx.ReadFile(proj)
	if err != nil {
		return Project{}, err
	}
	return readProjectFile(data, proj)
}

// readProjectFile returns a .NET Project object.
func readProjectFile(data []byte, proj string) (Project, error) {
	var p Project
	if err := xml.Unmarshal(data, &p); err != nil {
		return p, gcp.UserErrorf("unmarshalling %s: %v", proj, err)
	}
	return p, nil
}

// BuildableDir returns the directory of the provided GOOGLE_BUILDABLE env var.
// Buildable is in the form of app, app/app.csproj, or app/app.vbproj.
func BuildableDir() string {
	buildable := os.Getenv(env.Buildable)
	if strings.Contains(filepath.Ext(buildable), "proj") {
		return filepath.Dir(buildable)
	}
	return buildable
}

// RuntimeConfigJSONFiles returns all runtimeconfig.json files in 'path' (recursive).
// The runtimeconfig.json file is present for compiled .NET assemblies.
func RuntimeConfigJSONFiles(path string) ([]string, error) {
	var files []string
	var buildableDir = BuildableDir()
	if err := filepath.WalkDir(path, func(f string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(f, "runtimeconfig.json") && strings.HasPrefix(f, buildableDir) {
			files = append(files, f)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

// RuntimeConfigJSON matches the structure of a runtimeconfig.json file.
type RuntimeConfigJSON struct {
	RuntimeOptions runtimeOptions `json:"runtimeOptions"`
}

type framework struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type configProperties struct {
	SystemGCServer bool `json:"System.GC.Server"`
}

type runtimeOptions struct {
	TFM              string           `json:"tfm"`
	Framework        framework        `json:"framework"`
	Frameworks       []framework      `json:"frameworks"`
	ConfigProperties configProperties `json:"configProperties"`
}

// ReadRuntimeConfigJSON reads a given runtimeconfig.json file and returns a struct
// representation of the contents.
func ReadRuntimeConfigJSON(path string) (*RuntimeConfigJSON, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	var runCfg RuntimeConfigJSON
	if err := json.Unmarshal(bytes, &runCfg); err != nil {
		return nil, fmt.Errorf("unmarshalling %q to RuntimeConfig: %v", path, err)
	}
	return &runCfg, nil
}

// globalJSON represents the contents of a global.json file.
type globalJSON struct {
	Sdk struct {
		Version string `json:"version"`
	} `json:"sdk"`
}

// GetSDKVersion returns the appropriate .NET SDK version to use, with the following heuristic:
//  1. Return value of env variable GOOGLE_DOTNET_SDK_VERSION if present.
//  2. Return value of env variable GOOGLE_RUNTIME_VERSION if present.
//  3. Return SDK.Version from the .NET global.json file if present.
//  4. Return an empty string by default, which will cause us to use the latest version available
//     on dl.google.com (see runtime.InstallTarballIfNotCached for details).
func GetSDKVersion(ctx *gcp.Context) (string, error) {
	if version := os.Getenv(envSdkVersion); version != "" {
		ctx.Logf("Using .NET Core SDK version from %s: %s", envSdkVersion, version)
		return version, nil
	}
	if version := os.Getenv(env.RuntimeVersion); version != "" {
		ctx.Logf("Using .NET Core SDK version from %s: %s", env.RuntimeVersion, version)
		return version, nil
	}
	ctx.Logf("Looking for global.json in %v", ctx.ApplicationRoot())
	gjs, err := getGlobalJSONOrNil(ctx.ApplicationRoot())
	if err != nil {
		return "", err
	}
	if gjs != nil && gjs.Sdk.Version != "" {
		ctx.Logf("Using .NET Core SDK version from global.json: %s", gjs.Sdk.Version)
		return gjs.Sdk.Version, nil
	}
	ctx.Logf("Using latest stable .NET Core SDK version")
	return "", nil
}

func getGlobalJSONOrNil(applicationRoot string) (*globalJSON, error) {
	bytes, err := os.ReadFile(filepath.Join(applicationRoot, "global.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading global.json: %w", err)
	}
	var gjs globalJSON
	if err := json.Unmarshal(bytes, &gjs); err != nil {
		return nil, gcp.UserErrorf("unmarshalling global.json: %v", err)
	}
	return &gjs, nil
}

// AspNetRuntimeVersion determines what version of the ASP.NET core runtime should be installed by
// inspecting the GOOGLE_ASP_NET_CORE_VERSION environment variable. If it is not set it looks for
// a runtimeconfig.json the directory tree and extracts the version from it.
func AspNetRuntimeVersion(ctx *gcp.Context) (string, error) {
	if envVersion := os.Getenv(envAspNetCoreVersion); envVersion != "" {
		return envVersion, nil
	}
	rtCfgFiles, err := RuntimeConfigJSONFiles(".")
	if err != nil {
		return "", fmt.Errorf("finding runtimeconfig.json: %w", err)
	}
	// This invalid state should have been rejected by the detectFn already.
	if len(rtCfgFiles) == 0 {
		return "", fmt.Errorf("runtimeconfig.json does not exist")
	}
	runtimeVersion, err := GetRuntimeVersion(ctx, rtCfgFiles)
	if err != nil {
		return "", err
	}
	return runtimeVersion, nil
}

// FindProjectFile finds the csproj file using the 'GOOGLE_BUILDABLE' env var and falling back with a search of the current directory.
func FindProjectFile(ctx *gcp.Context) (string, error) {
	proj := os.Getenv(env.Buildable)
	if proj == "" {
		proj = "."
	}
	// Find the project file if proj is a directory.
	if fi, err := os.Stat(proj); os.IsNotExist(err) {
		return "", gcp.UserErrorf("%s does not exist", proj)
	} else if err != nil {
		return "", fmt.Errorf("stating %s: %v", proj, err)
	} else if fi.IsDir() {
		projFiles, err := ProjectFiles(ctx, proj)
		if err != nil {
			return "", err
		}
		if len(projFiles) != 1 {
			return "", gcp.UserErrorf("expected to find exactly one project file in directory %s, found %v", proj, projFiles)
		}
		proj = projFiles[0]
	}
	return proj, nil
}

// GetRuntimeVersion returns the Microsoft.AspNetCore.App version
// in the runtimeconfig.json file.
func GetRuntimeVersion(ctx *gcp.Context, rtCfgFiles []string) (string, error) {
	rtCfgFile, err := pickRuntimeConfigFile(ctx, rtCfgFiles)
	if err != nil {
		return "", err
	}

	rtCfg, err := ReadRuntimeConfigJSON(filepath.Join(ctx.ApplicationRoot(), rtCfgFile))
	if err != nil {
		return "", fmt.Errorf("reading runtimeconfig.json: %w", err)
	}
	if rtCfg.RuntimeOptions.Framework.Name == aspDotnetCore {
		return rtCfg.RuntimeOptions.Framework.Version, nil
	}
	for _, fw := range rtCfg.RuntimeOptions.Frameworks {
		if fw.Name == aspDotnetCore {
			return fw.Version, nil
		}
	}
	return "", fmt.Errorf("couldn't find runtime version from runtimeconfig.json: %#v", rtCfg)
}

func pickRuntimeConfigFile(ctx *gcp.Context, rtCfgFiles []string) (string, error) {
	if len(rtCfgFiles) == 0 {
		return "", fmt.Errorf("at least one 'runtimeconfig.json' is expected")
	}

	// This is needed for prebuilt app without ${BUILDABLE}.
	// To be backward compatible, we don't validate the path when there's only one rtCfgFile.
	if len(rtCfgFiles) == 1 {
		return rtCfgFiles[0], nil
	}

	rtCfgFile := ""
	buildable := os.Getenv(env.Buildable)
	buildableDir := BuildableDir()
	for _, f := range rtCfgFiles {
		ctx.Logf("Found runtimeconfig.json file: %v", f)
		if !strings.HasPrefix(f, buildable) {
			continue
		}

		if strings.HasPrefix(f, filepath.Join(buildableDir, "bin/Release")) {
			rtCfgFile = f
			break
		}
	}
	if rtCfgFile == "" {
		return "", fmt.Errorf("at least one 'runtimeconfig.json' under ${GOOGLE_BUILDABLE}=%q is expected",
			buildable)
	}
	ctx.Logf("Using runtimeconfig file %q", rtCfgFile)
	return rtCfgFile, nil
}
