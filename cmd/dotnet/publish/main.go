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

// Implements /bin/build for dotnet/publish buildpack.
package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	cacheTag = "prod dependencies"
)

// Metadata represents metadata stored for a packages layer.
type metadata struct {
	Version        string `toml:"version"`
	DependencyHash string `toml:"dependency_hash"`
}

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if _, exists := os.LookupEnv(env.Buildable); !exists && !ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.*proj") {
		ctx.OptOut("No project file found and %s not set.", env.Buildable)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	// Set the project file, getting it from the specified directory if relevant.
	proj := os.Getenv(env.Buildable)
	if proj == "" {
		proj = "."
	}
	if !strings.HasSuffix(proj, "proj") || filepath.Ext(proj) == "" {
		projFiles := ctx.Glob(filepath.Join(proj, "*.*proj"))
		if len(projFiles) != 1 {
			return gcp.UserErrorf("expected to find exactly one project file in directory %s, found %v", proj, projFiles)
		}
		proj = projFiles[0]
	}

	pkgLayer := ctx.Layer("packages")
	// We run a restore regardless (it generates some files into the application root that publish expects to find),
	// but we only need to do it on dependencies if we don't have a hot cache.
	restoreCmd := []string{"dotnet", "restore", "--packages", pkgLayer.Root, proj}

	ctx.Logf("Installing application dependencies.")
	cached, meta, err := checkCache(ctx, pkgLayer)
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	if cached {
		ctx.CacheHit(cacheTag)
		restoreCmd = append(restoreCmd, "--no-dependencies")
	} else {
		ctx.CacheMiss(cacheTag)
	}

	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: restoreCmd, Env: []string{"DOTNET_CLI_TELEMETRY_OPTOUT=true"}}, gcp.UserErrorKeepStderrTail)
	ctx.WriteMetadata(pkgLayer, &meta, layers.Build, layers.Cache)

	binLayer := ctx.Layer("bin")
	cmd := []string{
		"dotnet",
		"publish",
		"-nologo",
		"--verbosity", "minimal",
		"--configuration", "Release",
		"--output", "bin",
		"--no-restore",
		"--packages", pkgLayer.Root,
		proj,
	}

	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: cmd, Env: []string{"DOTNET_CLI_TELEMETRY_OPTOUT=true"}}, gcp.UserErrorKeepStderrTail)

	// Only calculate the entrypoint from this build if GOOGLE_ENTRYPOINT is not set, as we don't want to override that.
	entrypoint := os.Getenv(env.Entrypoint)
	if entrypoint == "" {
		ep, err := getEntrypoint(ctx, "bin", proj)
		if err != nil {
			return fmt.Errorf("getting entrypoint: %w", err)
		}
		entrypoint = strings.Join(ep, " ")
		ctx.OverrideBuildEnv(binLayer, env.Entrypoint, entrypoint)
	}
	ctx.WriteMetadata(binLayer, nil, layers.Build, layers.Launch)

	// Configure the entrypoint for production.
	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess([]string{"/bin/bash", "-c", "exec " + entrypoint})
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	ctx.AddWebProcess([]string{"dotnet", "watch", "--project", proj, "run"})
	devmode.AddSyncMetadata(ctx, devmode.DotNetSyncRules)

	return nil
}

// getEntrypoint retrieves the appropriate entrypoint for this build.
// To get the project filename, we check GOOGLE_BUILDABLE for a specified project file. If empty,
// we expect there to be a project file in the application root.
// With the project filename, we check the root output directory for a binary with the name of the
// project (e.g. app.csproj --> app).
// If not present, we check for a compiled library with the app name (e.g. app.csproj --> app.dll)
//and if present, we prepend the "dotnet run" command.
// If neither an executable binary or dll is present in the root output directory, we parse the
// project file for an AssemblyName field to see if the user changed the output name. If there is
// one, we check for the associated executable or dll file in the root output directory.
// If none of those files are present in the output directory, we return a user error.
func getEntrypoint(ctx *gcp.Context, root, proj string) ([]string, error) {
	ep := getEntrypointCmd(ctx, filepath.Join(root, strings.Split(filepath.Base(proj), ".")[0]))
	if ep != nil {
		return ep, nil
	}

	// If we didn't get anything from the default project file name, try to extract the output name from the project file.
	an, err := getAssemblyName(ctx, proj)
	if err != nil {
		return nil, fmt.Errorf("getting assembly name: %w", err)
	}
	ep = getEntrypointCmd(ctx, filepath.Join(root, an))
	if ep != nil {
		return ep, nil
	}

	// If we didn't get anything from that, something went wrong.
	return nil, gcp.UserErrorf("unable to find executable produced from %s, try using the default output name or the AssemblyName property", proj)
}

func getEntrypointCmd(ctx *gcp.Context, ep string) []string {
	// Check the default output, which is a binary named the same thing as the project file (minus the .*proj extension).
	if ctx.FileExists(ep) {
		return []string{ep}
	}

	// If not, check for a *.dll with that name. In this case, prefix the entrypoint with `dotnet run`.
	if ctx.FileExists(ep + ".dll") {
		return []string{"dotnet", "run", fmt.Sprintf("%s.dll", ep)}
	}

	return nil
}

func checkCache(ctx *gcp.Context, l *layers.Layer) (bool, *metadata, error) {
	// We cache all *.*proj files, as if we just cache just the main one, we would miss any changes
	// to other libraries implemented as part of the app. As many apps are structured such that the
	// main app only depends on the local binaries, that root project file would change very
	// infrequently while the associated library files would change significantly more often, as
	// that's where the primary implementation is done.
	projectFiles := strings.Split(ctx.Exec([]string{"find", ctx.ApplicationRoot(), "-name", "*.*proj"}).Stdout, "\n")
	globalJSON := filepath.Join(ctx.ApplicationRoot(), "global.json")
	if ctx.FileExists(globalJSON) {
		projectFiles = append(projectFiles, globalJSON)
	}
	currentVersion := ctx.Exec([]string{"dotnet", "--version"}).Stdout

	hash, err := gcp.DependencyHash(ctx, currentVersion, projectFiles...)
	if err != nil {
		return false, nil, fmt.Errorf("computing dependency hash: %w", err)
	}

	var meta metadata
	ctx.ReadMetadata(l, &meta)

	// Perform install, skipping if the dependency hash matches existing metadata.
	ctx.Debugf("Current dependency hash: %q", hash)
	ctx.Debugf("  Cache dependency hash: %q", meta.DependencyHash)
	if hash == meta.DependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, &meta, nil
	}

	if meta.DependencyHash == "" {
		ctx.Logf("No metadata found or no dependency_hash found on metadata. Unable to use cache layer.")
	}
	// Update the layer metadata.
	meta.DependencyHash = hash
	meta.Version = currentVersion
	return false, &meta, nil
}

// Project represents a .NET project file.
type Project struct {
	XMLName        xml.Name        `xml:"Project"`
	PropertyGroups []PropertyGroup `xml:"PropertyGroup"`
}

// PropertyGroup contains information about a project build.
type PropertyGroup struct {
	AssemblyName     string   `xml:"AssemblyName"`
	TargetFramework  []string `xml:"TargetFramework"`
	TargetFrameworks []string `xml:"TargetFrameworks"`
}

func getAssemblyName(ctx *gcp.Context, proj string) (string, error) {
	p, err := readProjectFile(ctx, proj)
	if err != nil {
		return "", fmt.Errorf("reading project file: %w", err)
	}
	an := ""
	for _, pg := range p.PropertyGroups {
		if pg.AssemblyName != "" {
			if an != "" {
				return "", gcp.UserErrorf("expected up to one AssemblyName defined, found multiple")
			}
			an = pg.AssemblyName
		}
	}
	return an, nil
}

func readProjectFile(ctx *gcp.Context, proj string) (Project, error) {
	data := ctx.ReadFile(proj)
	var p Project
	if err := xml.Unmarshal(data, &p); err != nil {
		return p, gcp.UserErrorf("unmarshalling %s: %v", proj, err)
	}
	return p, nil
}
