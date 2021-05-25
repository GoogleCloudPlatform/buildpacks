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

// Implements go/build buildpack.
// The build buildpack runs go build.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
)

const (
	noGoFileError         = "no Go files in"
	cannotFindModuleError = "cannot find module"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !ctx.HasAtLeastOne("*.go") {
		return gcp.OptOut("no .go files found"), nil
	}
	return gcp.OptIn("found .go files"), nil
}

func buildFn(ctx *gcp.Context) error {
	// Keep GOCACHE in Devmode for faster rebuilds.
	cl := ctx.Layer("gocache", gcp.BuildLayer, gcp.LaunchLayerIfDevMode)
	if devmode.Enabled(ctx) {
		cl.LaunchEnvironment.Override("GOCACHE", cl.Path)
	}

	// Create a layer for the compiled binary.  Add it to PATH in case
	// users wish to invoke the binary manually.
	bl := ctx.Layer("bin", gcp.LaunchLayer)
	bl.LaunchEnvironment.Prepend("PATH", string(os.PathListSeparator), bl.Path)
	outBin := filepath.Join(bl.Path, golang.OutBin)

	buildable, err := goBuildable(ctx)
	if err != nil {
		return fmt.Errorf("unable to find a valid buildable: %w", err)
	}

	// Build the application.
	bld := []string{"go", "build"}
	bld = append(bld, goBuildFlags()...)
	bld = append(bld, "-o", outBin)
	bld = append(bld, buildable)
	// BuildDirEnv should only be set by App Engine buildpacks.
	workdir := os.Getenv(golang.BuildDirEnv)
	if workdir == "" {
		workdir = ctx.ApplicationRoot()
	}
	ctx.Exec(bld, gcp.WithEnv("GOCACHE="+cl.Path), gcp.WithWorkDir(workdir), gcp.WithMessageProducer(printTipsAndKeepStderrTail(ctx)), gcp.WithUserAttribution)

	// Configure the entrypoint for production. Use the full path to save `skaffold debug`
	// from fetching the remote container image (tens to hundreds of megabytes), which is slow.
	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess([]string{outBin})
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	devmode.AddFileWatcherProcess(ctx, devmode.Config{
		BuildCmd: bld,
		RunCmd:   []string{outBin},
		Ext:      devmode.GoWatchedExtensions,
	})

	devmode.AddSyncMetadata(ctx, devmode.GoSyncRules)

	return nil
}

func goBuildable(ctx *gcp.Context) (string, error) {
	// The user tells us what to build.
	if buildable, ok := os.LookupEnv(env.Buildable); ok {
		return buildable, nil
	}

	// We have to guess which package/file to build.
	// `go build` will by default build the `.` package
	// but we try to be smarter by searching for a valid buildable.
	buildables, err := searchBuildables(ctx)
	if err != nil {
		return "", err
	}

	if len(buildables) == 1 {
		return buildables[0], nil
	}

	// Found no buildable or multiple buildables. Let Go build the default package.
	return ".", nil
}

// searchBuildables searches the source for all the files that contain
// a `main()` entrypoint.
func searchBuildables(ctx *gcp.Context) ([]string, error) {
	result := ctx.Exec([]string{"go", "list", "-f", `{{if eq .Name "main"}}{{.Dir}}{{end}}`, "./..."}, gcp.WithUserAttribution)

	var buildables []string

	for _, dir := range strings.Fields(result.Stdout) {
		rel, err := filepath.Rel(ctx.ApplicationRoot(), dir)
		if err != nil {
			return nil, fmt.Errorf("unable to find relative path for %q: %w", dir, err)
		}

		buildables = append(buildables, "./"+rel)
	}

	return buildables, nil
}

func goBuildFlags() []string {
	var flags []string
	if v := os.Getenv(env.GoGCFlags); v != "" {
		flags = append(flags, "-gcflags", v)
	}
	if v := os.Getenv(env.GoLDFlags); v != "" {
		flags = append(flags, "-ldflags", v)
	}
	return flags
}

func printTipsAndKeepStderrTail(ctx *gcp.Context) gcp.MessageProducer {
	return func(result *gcp.ExecResult) string {
		if result.ExitCode != 0 {
			// If `go build` fails with any of those two errors, there's a great chance
			// that we are not building the right package.
			if strings.Contains(result.Stderr, noGoFileError) || strings.Contains(result.Stderr, cannotFindModuleError) {
				ctx.Tipf("Tip: %q env var configures which Go package is built. Default is '.'", env.Buildable)
			}
		}

		return gcp.KeepStderrTail(result)
	}
}
