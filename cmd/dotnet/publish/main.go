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

// Implements dotnet/publish buildpack.
// The publish buildpack runs dotnet publish.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
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
	if _, exists := os.LookupEnv(env.Buildable); !exists && len(dotnet.ProjectFiles(ctx, ".")) == 0 {
		ctx.OptOut("no project file found and %s not set.", env.Buildable)
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	proj := os.Getenv(env.Buildable)
	if proj == "" {
		proj = "."
	}
	// Find the project file if proj is a directory.
	if fi, err := os.Stat(proj); os.IsNotExist(err) {
		return gcp.UserErrorf("%s does not exist", proj)
	} else if err != nil {
		return fmt.Errorf("stating %s: %v", proj, err)
	} else if fi.IsDir() {
		projFiles := dotnet.ProjectFiles(ctx, proj)
		if len(projFiles) != 1 {
			return gcp.UserErrorf("expected to find exactly one project file in directory %s, found %v", proj, projFiles)
		}
		proj = projFiles[0]
	}

	ctx.Logf("Installing application dependencies.")
	pkgLayer := ctx.Layer("packages")
	cached, meta, err := checkCache(ctx, pkgLayer)
	if err != nil {
		return fmt.Errorf("checking cache: %w", err)
	}
	// Print cache status for testing/debugging only, `dotnet restore` reuses any existing artifacts.
	if cached {
		ctx.CacheHit(cacheTag)
	} else {
		ctx.CacheMiss(cacheTag)
	}

	// Run restore regardless of cache status because it generates files expected by publish.
	cmd := []string{"dotnet", "restore", "--packages", pkgLayer.Root, proj}
	ctx.ExecUserWithParams(gcp.ExecParams{Cmd: cmd, Env: []string{"DOTNET_CLI_TELEMETRY_OPTOUT=true"}}, gcp.UserErrorKeepStderrTail)
	ctx.WriteMetadata(pkgLayer, &meta, layers.Build, layers.Cache)

	binLayer := ctx.Layer("bin")
	cmd = []string{
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

	// Infer the entrypoint in case an explicit override was not provided.
	entrypoint := os.Getenv(env.Entrypoint)
	if entrypoint == "" {
		ep, err := getEntrypoint(ctx, "bin", proj)
		if err != nil {
			return fmt.Errorf("getting entrypoint: %w", err)
		}
		entrypoint = strings.Join(ep, " ")
		ctx.DefaultBuildEnv(binLayer, env.Entrypoint, entrypoint)
	}
	ctx.DefaultLaunchEnv(binLayer, "DOTNET_RUNNING_IN_CONTAINER", "true")
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
// * Check the output directory for a binary or a libary with the same name as the project file (e.g. app.csproj --> app or app.dll).
// * If not found, parse the project file for an AssemblyName field and check for the associated binary or library file in the output directory.
// * If not found, return user error.
func getEntrypoint(ctx *gcp.Context, bin, proj string) ([]string, error) {
	ctx.Logf("Determining entrypoint from output directory %s and project file %s", bin, proj)
	p := strings.TrimSuffix(filepath.Base(proj), filepath.Ext(proj))
	ep := getEntrypointCmd(ctx, filepath.Join(bin, p))
	if ep != nil {
		return ep, nil
	}

	// If we didn't get anything from the default project file name, try to extract the output name from the project file.
	an, err := getAssemblyName(ctx, proj)
	if err != nil {
		return nil, fmt.Errorf("getting assembly name: %w", err)
	}
	ep = getEntrypointCmd(ctx, filepath.Join(bin, an))
	if ep != nil {
		return ep, nil
	}

	// If we didn't get anything from that, something went wrong.
	return nil, gcp.UserErrorf("unable to find executable produced from %s, try setting the AssemblyName property", proj)
}

func getEntrypointCmd(ctx *gcp.Context, ep string) []string {
	if ctx.FileExists(ep + ".dll") {
		return []string{"dotnet", fmt.Sprintf("%s.dll", ep)}
	}
	return nil
}

func checkCache(ctx *gcp.Context, l *layers.Layer) (bool, *metadata, error) {
	// We cache all *.*proj files, as if we just cache just the main one, we would miss any changes
	// to other libraries implemented as part of the app. As many apps are structured such that the
	// main app only depends on the local binaries, that root project file would change very
	// infrequently while the associated library files would change significantly more often, as
	// that's where the primary implementation is done.
	projectFiles := dotnet.ProjectFiles(ctx, ".")
	globalJSON := filepath.Join(ctx.ApplicationRoot(), "global.json")
	if ctx.FileExists(globalJSON) {
		projectFiles = append(projectFiles, globalJSON)
	}
	currentVersion := ctx.Exec([]string{"dotnet", "--version"}).Stdout

	hash, err := cache.Hash(ctx, cache.WithStrings(currentVersion), cache.WithFiles(projectFiles...))
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
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	// Update the layer metadata.
	meta.DependencyHash = hash
	meta.Version = currentVersion
	return false, &meta, nil
}

func getAssemblyName(ctx *gcp.Context, proj string) (string, error) {
	p, err := dotnet.ReadProjectFile(ctx, proj)
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
