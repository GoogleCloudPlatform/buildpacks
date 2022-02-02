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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// See getSDKChannelForTargetFramework for how this map is used.
	frameworkVersionToSDKVersion = map[string]string{
		"netcoreapp1.0": "1.0",
		"netcoreapp2.0": "2.2",
		"netcoreapp2.1": "2.2",
		"netcoreapp2.2": "2.2",
		"netcoreapp3.0": "3.1",
		"netcoreapp3.1": "3.1",
	}
)

// ProjectFiles finds all project files supported by dotnet.
func ProjectFiles(ctx *gcp.Context, dir string) []string {
	result := ctx.Exec([]string{"find", dir, "-regex", `.*\.\(cs\|fs\|vb\)proj`}, gcp.WithUserTimingAttribution).Stdout
	result = strings.TrimSpace(result)
	if result == "" {
		return nil
	}
	return strings.Split(result, "\n")
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
	data := ctx.ReadFile(proj)
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

// RuntimeConfigJSONFiles returns all runtimeconfig.json files in 'path' (recursive).
// The runtimeconfig.json file is present for compiled .NET assemblies.
func RuntimeConfigJSONFiles(path string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(path, func(f string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(f, "runtimeconfig.json") {
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

// getSDKChannelForTargetFramework returns the appropriate SDK channel for the given target framework.
// The purpose of this function is to select an older SDK channel for projects which have a
// TargetFramework which is not supported by latest. The second, "ok", value is false when there is
// no appropriate match, signifying that the caller should use the latest framework version.
//
// Note that the .NET 5.0 SDK can compile projects that target 1.0 -> 5.0, however, the 5.0 runtime
// cannot run the app. For .NET, a runtime can run any app within the same major version. We only
// install the SDK into the build layer and then only install the runtime into the launch layer.
// Since we install the runtime version associated with an SDK then we should pick an SDK which
// will have an associated runtime version that is compatible with the target framework. We choose
// the greatest compatible SDK/runtime combo to take advantage of an security fixes that have been
// made. For example, if the app is targetting netcoreapp3.0 then we return "3.1" as the 3.1
// runtime can run the app.
func getSDKChannelForTargetFramework(tfm string) (string, bool) {
	value, ok := frameworkVersionToSDKVersion[tfm]
	if ok {
		return value, ok
	}
	// The 'current' format for recent release of the .NET SDK is 'net5.0', 'net6.0', etc.
	netPrefix := "net"
	if strings.HasPrefix(tfm, netPrefix) {
		return strings.TrimPrefix(tfm, netPrefix), true
	}
	return "", false
}

// GetSDKVersion returns the appropriate .NET SDK version to use, with the following heuristic:
// 1. Return value of env variable GOOGLE_RUNTIME_VERSION if present.
// 2. Return SDK.Version from the .NET global.json file if present.
// 3. Search for runtimeconfig.json, if present, use the target framework  value in
//    runtimeOptions.tfm and use the latest compatible SDK version.
// 3. Get the first target framework version from the Project (csproj) and use the latest
//    compatible SDK version.
// 4. Query for the latest LTS version of the SDK via azure web service and return result.
func GetSDKVersion(ctx *gcp.Context) (string, error) {
	version, ok, err := lookupSpecifiedSDKVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("looking up runtime version: %w", err)
	}
	if ok {
		return version, nil
	}
	sdkChannel, err := getSDKChannel(ctx)
	if err != nil {
		return "", err
	}
	sdkVersion, err := client.New().GetLatestSDKVersionForChannel(sdkChannel)
	if err != nil {
		return "", gcp.UserErrorf("getting latest version for channel %q: %v", sdkChannel, err)
	}
	return sdkVersion, nil
}

func getSDKChannel(ctx *gcp.Context) (string, error) {
	rtCfgFiles, err := RuntimeConfigJSONFiles(".")
	if err != nil {
		return "", fmt.Errorf("finding runtimeconfig.json: %w", err)
	}
	if len(rtCfgFiles) > 1 {
		return "", fmt.Errorf("more than one runtimeconfig.json file found: %v", rtCfgFiles)
	}
	if len(rtCfgFiles) == 1 {
		rtCfg, err := ReadRuntimeConfigJSON(rtCfgFiles[0])
		if err != nil {
			return "", fmt.Errorf("reading runtimeconfig.json: %w", err)
		}
		sdkChannel, ok := getSDKChannelForTargetFramework(rtCfg.RuntimeOptions.TFM)
		if !ok {
			return "", fmt.Errorf("cannot use runtimeconfig.json tfm value of %q to get .NET sdk version", rtCfg.RuntimeOptions.TFM)
		}
		return sdkChannel, nil
	}
	projPath, err := FindProjectFile(ctx)
	if err != nil {
		return "", fmt.Errorf("finding project: %w", err)
	}
	project, err := ReadProjectFile(ctx, projPath)
	if err != nil {
		return "", fmt.Errorf("reading project at %q: %w", projPath, err)
	}
	return getSDKChannelForProject(project)
}

func getSDKChannelForProject(p Project) (string, error) {
	if len(p.PropertyGroups) == 0 {
		return "", fmt.Errorf("cannot detect SDK version: project's TargetFrameworks field is empty")
	}
	version, ok := getSDKChannelForTargetFramework(p.PropertyGroups[0].TargetFramework)
	if !ok {
		return "", fmt.Errorf("cannot use project's TargetFramework value of %q to get .NET sdk version", p.PropertyGroups[0].TargetFramework)
	}
	return version, nil
}

// lookupSpecifiedSDKVersion returns the SDK version specified in the env var GOOGLE_RUNTIME_VERSION *or*
// the .NET global.json file. If no such version is specified, then the second return value is false.
func lookupSpecifiedSDKVersion(ctx *gcp.Context) (string, bool, error) {
	version := os.Getenv(env.RuntimeVersion)
	if version != "" {
		ctx.Logf("Using .NET Core SDK version from env: %s", version)
		return version, true, nil
	}
	ctx.Logf("Looking for global.json in %v", ctx.ApplicationRoot())
	gjs, err := getGlobalJSONOrNil(ctx.ApplicationRoot())
	if err != nil || gjs == nil {
		return "", false, err
	}
	if gjs.Sdk.Version == "" {
		return "", false, nil
	}
	ctx.Logf("Using .NET Core SDK version from global.json: %s", version)
	return gjs.Sdk.Version, true, nil
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
		projFiles := ProjectFiles(ctx, proj)
		if len(projFiles) != 1 {
			return "", gcp.UserErrorf("expected to find exactly one project file in directory %s, found %v", proj, projFiles)
		}
		proj = projFiles[0]
	}
	return proj, nil
}
