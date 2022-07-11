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
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/cache"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	cacheTag          = "prod dependencies"
	dependencyHashKey = "dependency_hash"
	versionKey        = "version"
	outputDirectory   = "bin"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if _, exists := os.LookupEnv(env.Buildable); exists {
		return gcp.OptInEnvSet(env.Buildable), nil
	}
	files, err := dotnet.ProjectFiles(ctx, ".")
	if err != nil {
		return nil, err
	}
	if len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}

	return gcp.OptOut(fmt.Sprintf("no project files found and %s not set", env.Buildable)), nil
}

func buildFn(ctx *gcp.Context) error {
	proj, err := dotnet.FindProjectFile(ctx)
	if err != nil {
		return fmt.Errorf("finding project: %w", err)
	}
	ctx.Logf("Installing application dependencies.")
	pkgLayer, err := ctx.Layer("packages", gcp.BuildLayer, gcp.CacheLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	cached, err := checkCache(ctx, pkgLayer)
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
	cmd := []string{"dotnet", "restore", "--packages", pkgLayer.Path, proj}
	if _, err := ctx.ExecWithErr(cmd, gcp.WithEnv("DOTNET_CLI_TELEMETRY_OPTOUT=true"), gcp.WithUserAttribution); err != nil {
		return err
	}

	binLayer, err := ctx.Layer("bin", gcp.BuildLayer, gcp.LaunchLayer)
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	cmd = []string{
		"dotnet",
		"publish",
		"-nologo",
		"--verbosity", "minimal",
		"--configuration", "Release",
		"--output", outputDirectory,
		"--no-restore",
		"--packages", pkgLayer.Path,
		proj,
	}

	if args := os.Getenv(env.BuildArgs); args != "" {
		// Use bash to excute the command to avoid havnig to parse the build arguments.
		// strings.Fields may be unsafe here in case some arguments have a space.
		cmd = []string{"/bin/bash", "-c", strings.Join(append(cmd, args), " ")}
	}

	if _, err := ctx.ExecWithErr(cmd, gcp.WithEnv("DOTNET_CLI_TELEMETRY_OPTOUT=true"), gcp.WithUserAttribution); err != nil {
		return err
	}

	// Infer the entrypoint in case an explicit override was not provided.
	entrypoint := os.Getenv(env.Entrypoint)
	if entrypoint != "" {
		entrypoint = "exec " + entrypoint
	} else {
		ep, err := getEntrypoint(ctx, outputDirectory, proj)
		if err != nil {
			return fmt.Errorf("getting entrypoint: %w", err)
		}
		entrypoint = ep
		binLayer.BuildEnvironment.Default(env.Entrypoint, entrypoint)
	}
	binLayer.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")

	// Configure the entrypoint for production.
	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess([]string{"/bin/bash", "-c", entrypoint})
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	ctx.AddWebProcess([]string{"dotnet", "watch", "--project", proj, "run"})
	devmode.AddSyncMetadata(ctx, devmode.DotNetSyncRules)
	return nil
}

// getEntrypoint retrieves the appropriate entrypoint for this build.
// * Check the output directory for a binary or a library with the same name as the project file (e.g. app.csproj --> app or app.dll).
// * If not found, parse the project file for an AssemblyName field and check for the associated binary or library file in the output directory.
// * If not found, return user error.
func getEntrypoint(ctx *gcp.Context, bin, proj string) (string, error) {
	ctx.Logf("Determining entrypoint from output directory %s and project file %s", bin, proj)
	p := strings.TrimSuffix(filepath.Base(proj), filepath.Ext(proj))

	ep, err := getEntrypointCmd(ctx, filepath.Join(bin, p))
	if err != nil {
		return "", err
	}
	if ep != "" {
		return ep, nil
	}

	// If we didn't get anything from the default project file name, try to extract the output name from the project file.
	an, err := getAssemblyName(ctx, proj)
	if err != nil {
		return "", fmt.Errorf("getting assembly name: %w", err)
	}
	ep, err = getEntrypointCmd(ctx, filepath.Join(bin, an))
	if err != nil {
		return "", err
	}
	if ep != "" {
		return ep, nil
	}

	// If we didn't get anything from that, something went wrong.
	return "", gcp.UserErrorf("unable to find executable produced from %s, try setting the AssemblyName property", proj)
}

func getEntrypointCmd(ctx *gcp.Context, ep string) (string, error) {
	dll := ep + ".dll"
	dllExists, err := ctx.FileExists(dll)
	if err != nil {
		return "", err
	}
	if dllExists {
		return fmt.Sprintf("cd %s && exec dotnet %s", path.Dir(dll), path.Base(dll)), nil
	}
	return "", nil
}

func checkCache(ctx *gcp.Context, l *libcnb.Layer) (bool, error) {
	// We cache all *.*proj files, as if we just cache just the main one, we would miss any changes
	// to other libraries implemented as part of the app. As many apps are structured such that the
	// main app only depends on the local binaries, that root project file would change very
	// infrequently while the associated library files would change significantly more often, as
	// that's where the primary implementation is done.
	projectFiles, err := dotnet.ProjectFiles(ctx, ".")
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
	result, err := ctx.ExecWithErr([]string{"dotnet", "--version"})
	if err != nil {
		return false, err
	}
	currentVersion := result.Stdout

	hash, err := cache.Hash(ctx, cache.WithStrings(currentVersion), cache.WithFiles(projectFiles...))
	if err != nil {
		return false, fmt.Errorf("computing dependency hash: %w", err)
	}

	// Perform install, skipping if the dependency hash matches existing metadata.
	metaDependencyHash := ctx.GetMetadata(l, dependencyHashKey)
	ctx.Debugf("Current dependency hash: %q", hash)
	ctx.Debugf("  Cache dependency hash: %q", metaDependencyHash)
	if hash == metaDependencyHash {
		ctx.Logf("Dependencies cache hit, skipping installation.")
		return true, nil
	}

	if metaDependencyHash == "" {
		ctx.Debugf("No metadata found from a previous build, skipping cache.")
	}
	// Update the layer metadata.
	ctx.SetMetadata(l, dependencyHashKey, hash)
	ctx.SetMetadata(l, versionKey, currentVersion)
	return false, nil
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
