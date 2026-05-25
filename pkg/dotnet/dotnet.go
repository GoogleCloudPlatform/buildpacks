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
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb/v2"
)

const (
	aspDotnetCore = "Microsoft.AspNetCore.App"
	envSdkVersion = "GOOGLE_DOTNET_SDK_VERSION"
	googleMin22   = "google.min.22"
	// EnvRuntimeVersion is the environment variable key for storing the target dotnet runtime version.
	EnvRuntimeVersion = "GOOGLE_ASP_NET_CORE_VERSION"
	// PublishLayerName is the name of the directory containing the publish layer
	PublishLayerName = "publish"
	// PublishOutputDirName is passed as the output directory for `dotnet publish`.
	PublishOutputDirName = "bin"
)

// SkipEnvVariablesAssignmentCapability is the capability key for skipping runtime environment variables assignment.
const SkipEnvVariablesAssignmentCapability = "dotnet.SkipEnvVariablesAssignmentCapability"

// MakerSkipEnvVariablesAssignment implements the SkipEnvVariablesAssignment interface for the maker tool.
type MakerSkipEnvVariablesAssignment struct{}

// SkipVariables skips the launch environment variables setup.
func (m MakerSkipEnvVariablesAssignment) SkipVariables(ctx *gcp.Context, rtl *libcnb.Layer) error {
	return nil
}

// SkipEnvVariablesAssignment is an interface for skipping runtime environment variables assignment.
type SkipEnvVariablesAssignment interface {
	SkipVariables(ctx *gcp.Context, rtl *libcnb.Layer) error
}

// PublisherCapability is the capability key for the maker Dotnet publisher.
const PublisherCapability = "dotnet.PublisherCapability"

// Publisher is an interface for restoring and publishing Dotnet applications in maker mode.
type Publisher interface {
	Publish(ctx *gcp.Context, proj, buildArgs string) error
}

// MakerDotnetPublisher implements the Publisher interface for the maker tool.
type MakerDotnetPublisher struct{}

// Publish restores, publishes, and determines the runtime entrypoint of the Dotnet application in maker mode.
func (p MakerDotnetPublisher) Publish(ctx *gcp.Context, proj, buildArgs string) error {
	return Publish(ctx, proj, buildArgs, false)
}

const (
	cacheTag          = "prod dependencies"
	dependencyHashKey = "dependency_hash"
	versionKey        = "version"
)

// Publish restores, publishes, and determines the runtime entrypoint of the Dotnet application.
// If useLayer is true, it performs the publish steps using GCP layers and devmode setup.
// If useLayer is false (e.g. MakerMode), it publishes directly to the application root without layers.
func Publish(ctx *gcp.Context, proj, buildArgs string, useLayer bool) error {
	var outputDirectory string
	var pkgLayer *libcnb.Layer
	var binLayer *libcnb.Layer
	var err error

	if useLayer {
		ctx.Logf("Installing application dependencies.")
		pkgLayer, err = ctx.Layer("packages", gcp.BuildLayer, gcp.CacheLayer)
		if err != nil {
			return fmt.Errorf("creating layer: %w", err)
		}

		cached, err := checkCache(ctx, pkgLayer)
		if err != nil {
			return fmt.Errorf("checking cache: %w", err)
		}
		if cached {
			ctx.CacheHit(cacheTag)
		} else {
			ctx.CacheMiss(cacheTag)
		}

		binLayer, err = ctx.Layer(PublishLayerName, gcp.BuildLayer, gcp.LaunchLayer)
		if err != nil {
			return fmt.Errorf("creating layer: %w", err)
		}

		outputDirectory = filepath.Join(binLayer.Path, PublishOutputDirName)

		// The existence of a project file indicates this is not prebuilt. Any uploaded bin folder interferes with publish.
		deleted, err := deleteFolder(ctx, filepath.Join(ctx.ApplicationRoot(), PublishOutputDirName))
		if err != nil {
			return fmt.Errorf("deleting upload bin: %w", err)
		}
		if deleted {
			ctx.Warnf("A project file was uploaded, causing `dotnet publish` to be called, but the output bin folder already existed in application source. Deleting %v.", outputDirectory)
		}
	} else {
		outputDirectory = filepath.Join(ctx.ApplicationRoot(), PublishOutputDirName)

		globalJSON := filepath.Join(ctx.ApplicationRoot(), "global.json")
		globalJSONBak := filepath.Join(ctx.ApplicationRoot(), "global.json.bak")

		globalJSONExists, err := ctx.FileExists(globalJSON)
		if err == nil && globalJSONExists {
			ctx.Logf("Temporarily renaming global.json to global.json.bak to roll forward SDK build.")
			if err := os.Rename(globalJSON, globalJSONBak); err != nil {
				return fmt.Errorf("renaming global.json: %w", err)
			}
			defer func() {
				ctx.Logf("Restoring global.json from global.json.bak.")
				os.Rename(globalJSONBak, globalJSON)
			}()
		}
	}

	// 1. Restore
	restoreCmd := []string{"dotnet", "restore"}
	if useLayer {
		restoreCmd = append(restoreCmd, "--packages", pkgLayer.Path)
	}
	restoreCmd = append(restoreCmd, proj)

	if _, err := ctx.Exec(restoreCmd, gcp.WithEnv("DOTNET_CLI_TELEMETRY_OPTOUT=true"), gcp.WithUserAttribution); err != nil {
		return err
	}

	// 2. Publish
	publishCmd := []string{
		"dotnet",
		"publish",
		"-nologo",
		"--verbosity", "minimal",
		"--configuration", "Release",
		"--output", outputDirectory,
		"--no-restore",
	}
	if useLayer {
		publishCmd = append(publishCmd, "--packages", pkgLayer.Path)
	}
	publishCmd = append(publishCmd, proj)

	if buildArgs != "" {
		// Use bash to execute the command to avoid having to parse the build arguments.
		// strings.Fields may be unsafe here in case some arguments have a space.
		publishCmd = []string{"/bin/bash", "-c", strings.Join(append(publishCmd, buildArgs), " ")}
	}

	if _, err := ctx.Exec(publishCmd, gcp.WithEnv("DOTNET_CLI_TELEMETRY_OPTOUT=true"), gcp.WithUserAttribution); err != nil {
		return err
	}

	// 3. Runtime Version
	runtimeVersion, err := GetRuntimeVersion(ctx, outputDirectory)
	if err != nil {
		return fmt.Errorf("getting runtime version: %w", err)
	}

	if useLayer {
		binLayer.BuildEnvironment.Default(EnvRuntimeVersion, runtimeVersion)
	}

	// 4. Symlink (only for layers)
	if useLayer {
		if err := configureBinSymlink(ctx, outputDirectory); err != nil {
			return fmt.Errorf("creating symlink: %w", err)
		}
	}

	// 5. Entrypoint
	entrypoint := os.Getenv(env.Entrypoint)
	if entrypoint != "" {
		entrypoint = "exec " + entrypoint
	} else {
		ep, err := Entrypoint(ctx, outputDirectory, proj)
		if err != nil {
			return fmt.Errorf("getting entrypoint: %w", err)
		}
		entrypoint = ep
		if useLayer {
			binLayer.BuildEnvironment.Default(env.Entrypoint, entrypoint)
		}
	}

	if useLayer {
		binLayer.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")

		// Configure the entrypoint for production.
		if !devmode.Enabled(ctx) {
			ctx.AddWebProcess([]string{"/bin/bash", "-c", entrypoint})
			return nil
		}

		// Configure the entrypoint and metadata for dev mode.
		ctx.AddWebProcess([]string{"dotnet", "watch", "--project", proj, "run"})
		return nil
	}

	// MakerMode (useLayer == false)
	ctx.AddWebProcess([]string{"/bin/bash", "-c", entrypoint})
	return nil
}

func checkCache(ctx *gcp.Context, l *libcnb.Layer) (bool, error) {
	projectFiles, err := ProjectFiles(ctx, ".")
	if err != nil {
		return false, err
	}
	globalJSON := filepath.Join(ctx.ApplicationRoot(), "global.json")
	globalJSONExists, err := ctx.FileExists(globalJSON)
	if err != nil {
		return false, err
	}
	if globalJSONExists {
		projectFiles = append(projectFiles, globalJSON)
	}
	result, err := ctx.Exec([]string{"dotnet", "--version"})
	if err != nil {
		return false, err
	}
	currentVersion := result.Stdout

	hash, cached, err := cache.HashAndCheck(ctx, l, dependencyHashKey,
		cache.WithStrings(currentVersion),
		cache.WithFiles(projectFiles...))
	if err != nil {
		return false, err
	}

	if cached {
		return true, nil
	}

	cache.Add(ctx, l, dependencyHashKey, hash)
	ctx.SetMetadata(l, versionKey, currentVersion)
	return false, nil
}

// deleteFolder returns whether the folder was deleted
func deleteFolder(ctx *gcp.Context, folder string) (bool, error) {
	exists, err := ctx.FileExists(folder)
	if err != nil {
		return false, err
	}
	if exists {
		if err := os.RemoveAll(folder); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func configureBinSymlink(ctx *gcp.Context, binLayerPath string) error {
	linkTarget := filepath.Join(ctx.ApplicationRoot(), PublishOutputDirName)

	if deleted, err := deleteFolder(ctx, linkTarget); err != nil {
		return fmt.Errorf("deleting %s: %v", linkTarget, err)
	} else if deleted {
		ctx.Warnf("Deleted folder: %v", linkTarget)
	} else {
		ctx.Warnf("Not deleting folder: %v", linkTarget)
	}

	if err := os.Symlink(binLayerPath, linkTarget); err != nil {
		return fmt.Errorf("linking %s: %v", binLayerPath, err)
	}
	return nil
}

// AssemblyName retrieves the assembly name property from .csproj file.
func AssemblyName(ctx *gcp.Context, proj string) (string, error) {
	p, err := ReadProjectFile(ctx, proj)
	if err != nil {
		return "", fmt.Errorf("reading project file: %w", err)
	}
	var assemblyNames []string
	for _, pg := range p.PropertyGroups {
		if pg.AssemblyName != "" {
			assemblyNames = append(assemblyNames, pg.AssemblyName)
		}
	}
	if len(assemblyNames) != 1 {
		return "", gcp.UserErrorf("expected exactly one AssemblyName, found %v", assemblyNames)
	}
	return assemblyNames[0], nil
}

// Entrypoint retrieves the appropriate entrypoint command for the direct maker build.
func Entrypoint(ctx *gcp.Context, bin, proj string) (string, error) {
	ctx.Logf("Determining entrypoint from output directory %s and project file %s", bin, proj)
	p := strings.TrimSuffix(filepath.Base(proj), filepath.Ext(proj))

	ep, err := EntrypointCmd(ctx, filepath.Join(bin, p))
	if err != nil {
		return "", err
	}
	if ep != "" {
		return ep, nil
	}

	an, err := AssemblyName(ctx, proj)
	if err != nil {
		return "", fmt.Errorf("getting assembly name: %w", err)
	}
	ep, err = EntrypointCmd(ctx, filepath.Join(bin, an))
	if err != nil {
		return "", err
	}
	if ep != "" {
		return ep, nil
	}

	return "", gcp.UserErrorf("unable to find executable produced from %s, try setting the AssemblyName property", proj)
}

// EntrypointCmd constructs direct relative execution string for .dll targets.
func EntrypointCmd(ctx *gcp.Context, ep string) (string, error) {
	dll := ep + ".dll"
	dllExists, err := ctx.FileExists(dll)
	if err != nil {
		return "", err
	}
	if dllExists {
		dir := filepath.Dir(dll)
		if rel, err := filepath.Rel(ctx.ApplicationRoot(), dir); err == nil && !strings.HasPrefix(rel, "..") {
			return fmt.Sprintf("exec dotnet %s", filepath.Join(rel, filepath.Base(dll))), nil
		}
		return fmt.Sprintf("cd %s && exec dotnet %s", dir, filepath.Base(dll)), nil
	}
	return "", nil
}

var (
	// latestDotnetSDKVersionPerStack is the latest .NET version per stack to use if not specified by the user.
	latestDotnetSDKVersionPerStack = map[string]string{
		runtime.Ubuntu2204: "8.*.*",
		runtime.Ubuntu2404: "10.*.*",
	}
	projRe = regexp.MustCompile(`(?i)\.(cs|fs|vb)proj$`)
)

// ProjectFiles finds all project files supported by dotnet.
func ProjectFiles(ctx *gcp.Context, dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && projRe.MatchString(d.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
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

// RuntimeConfigJSONFiles returns all runtimeconfig.json files in 'path'.
// The runtimeconfig.json file is present for compiled .NET assemblies.
func RuntimeConfigJSONFiles(path string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(path, "*runtimeconfig.json"))
	if err != nil {
		return nil, err
	}
	if files == nil {
		return []string{}, nil
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
//  4. If none of above are present, return the the latest SDK version available for the stack being used.
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

	os := runtime.OSForStack(ctx)

	version, ok := latestDotnetSDKVersionPerStack[os]
	if !ok {
		return "", gcp.UserErrorf("invalid stack for .NET runtime: %q. Please use a supported stack", os)
	}

	ctx.Logf(".NET SDK version not specified, using the latest available .NET SDK for the stack %q", os)
	return version, nil
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

// GetRuntimeVersion returns the value in GOOGLE_ASP_NET_CORE_VERSION, and if not set, returns
// Microsoft.AspNetCore.App version in the runtimeconfig.json file found in dir.
func GetRuntimeVersion(ctx *gcp.Context, dir string) (string, error) {
	envVarVersion := os.Getenv(EnvRuntimeVersion)
	if envVarVersion != "" {
		ctx.Logf("Determined runtime version from %v: %v", EnvRuntimeVersion, envVarVersion)
		return envVarVersion, nil
	}

	rtCfgVersion, rtCfgFile, rtCfgErr := getRuntimeVersionFromRtCfgDir(ctx, dir)
	if rtCfgErr != nil {
		return "", fmt.Errorf("%v was not set; when %v absent, getting version from runtimeconfig.json failed: %w", EnvRuntimeVersion, EnvRuntimeVersion, rtCfgErr)
	}
	ctx.Logf("Determined runtime version from %v: %v", rtCfgFile, rtCfgVersion)
	return rtCfgVersion, nil
}

func getRuntimeVersionFromRtCfgDir(ctx *gcp.Context, dir string) (string, string, error) {
	rtCfgFiles, err := RuntimeConfigJSONFiles(dir)
	if err != nil {
		return "", "", gcp.InternalErrorf("finding runtimeconfig.json: %v", err)
	}
	if len(rtCfgFiles) > 1 {
		return "", "", fmt.Errorf("more than one runtimeconfig.json file found: %v", rtCfgFiles)
	}

	if len(rtCfgFiles) < 1 {
		return "", "", fmt.Errorf("no runtimeconfig.json file was found")
	}
	ctx.Logf("Found runtimeconfig file %q", rtCfgFiles[0])

	version := ""
	rtCfg, err := ReadRuntimeConfigJSON(rtCfgFiles[0])
	if err != nil {
		return "", rtCfgFiles[0], fmt.Errorf("reading runtimeconfig.json: %w", err)
	}

	if rtCfg.RuntimeOptions.Framework.Name == aspDotnetCore {
		version = rtCfg.RuntimeOptions.Framework.Version
	} else {
		for _, fw := range rtCfg.RuntimeOptions.Frameworks {
			if fw.Name == aspDotnetCore {
				version = fw.Version
				break
			}
		}
	}

	if version == "" {
		return "", rtCfgFiles[0], fmt.Errorf("couldn't find runtime version for framework %s from "+
			"runtimeconfig.json: %#v", aspDotnetCore, rtCfg)
	}

	return version, rtCfgFiles[0], nil
}

// RequiresGlobalizationInvariant returns true if the system lacks the OS packages necessary to
// support .NET globalization.
func RequiresGlobalizationInvariant(ctx *gcp.Context) bool {
	return ctx.StackID() == googleMin22
}
