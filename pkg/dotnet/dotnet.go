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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
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

// globalJSON represents the contents of a global.json file.
type globalJSON struct {
	Sdk struct {
		Version string `json:"version"`
	} `json:"sdk"`
}

// GetSDKVersion returns the appropriate .NET SDK version to use, with the following heuristic:
// 1. Return value of env variable GOOGLE_RUNTIME_VERSION if present
// 2. Return SDK.Version from the .NET global.json file if present
// 3. Query web service at `versionURL` and return result
func GetSDKVersion(ctx *gcp.Context) (string, error) {
	version, ok, err := lookupSpecifiedSDKVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("looking up runtime version: %w", err)
	}
	if ok {
		return version, nil
	}
	// Use the latest LTS version.
	version, err = client.GetLatestSDKVersion()
	if err != nil {
		return "", gcp.UserErrorf("getting latest version: %v", err)
	}
	ctx.Logf("Using the latest LTS version of .NET SDK: %s", version)
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
