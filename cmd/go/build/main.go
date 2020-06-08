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
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/devmode"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/golang"
	"github.com/buildpack/libbuildpack/layers"
)

const (
	noGoFileError         = "no Go files in"
	cannotFindModuleError = "cannot find module"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if !ctx.HasAtLeastOne("*.go") {
		ctx.OptOut("No *.go files found")
	}
	return nil
}

func buildFn(ctx *gcp.Context) error {
	// Create a cached layer for the GOCACHE.
	cl := ctx.Layer("gocache")
	lf := []layers.Flag{layers.Cache, layers.Build}
	if devmode.Enabled(ctx) {
		lf = append(lf, layers.Launch)
		ctx.OverrideLaunchEnv(cl, "GOCACHE", cl.Root)
	}
	ctx.WriteMetadata(cl, nil, lf...)

	// Create a layer for the compiled binary.  Add it to PATH in case
	// users wish to invoke the binary manually.
	bl := ctx.Layer("bin")
	ctx.PrependPathLaunchEnv(bl, "PATH", bl.Root)
	ctx.WriteMetadata(bl, nil, layers.Launch)
	outBin := filepath.Join(bl.Root, golang.OutBin)

	buildable := goBuildable(ctx)

	// Build the application.
	cmd := []string{"go", "build"}
	cmd = append(cmd, goBuildFlags()...)
	cmd = append(cmd, "-o", outBin)
	cmd = append(cmd, buildable)
	ctx.ExecUserWithParams(gcp.ExecParams{
		Cmd: cmd,
		Env: []string{"GOCACHE=" + cl.Root},
	}, printTipsAndKeepStderrTail(ctx))

	// Configure the entrypoint for production.  Use the full path to save `skaffold debug`
	// from fetching the remote container image (tens to hundreds of megabytes), which is slow.
	if !devmode.Enabled(ctx) {
		ctx.AddWebProcess([]string{outBin})
		return nil
	}

	// Configure the entrypoint and metadata for dev mode.
	cmd = []string{"go", "run"}
	cmd = append(cmd, goBuildFlags()...)
	cmd = append(cmd, buildable)
	devmode.AddFileWatcherProcess(ctx, devmode.Config{
		Cmd: cmd,
		Ext: devmode.GoWatchedExtensions,
	})
	devmode.AddSyncMetadata(ctx, devmode.GoSyncRules)

	return nil
}

func goBuildable(ctx *gcp.Context) string {
	// The user tells us what to build.
	if buildable, ok := os.LookupEnv(env.Buildable); ok {
		return buildable
	}

	// We have to guess which package/file to build.
	// `go build` will by default build the `.` package
	// but we try to be smarter by searching for a valid buildable.
	buildables := searchBuildables(ctx)
	if len(buildables) == 1 {
		return buildables[0]
	}

	// Found no buildable or multiple buildables. Let Go build the default package.
	return "."
}

// searchBuildables searches the source for all the files that contain
// a `main()` entrypoint.
func searchBuildables(ctx *gcp.Context) []string {
	result := ctx.Exec([]string{"go", "list", "-f", `{{if eq .Name "main"}}{{.Dir}}{{end}}`, "./..."})

	var buildables []string

	for _, dir := range strings.Fields(result.Stdout) {
		rel, err := filepath.Rel(ctx.ApplicationRoot(), dir)
		if err != nil {
			ctx.Exit(1, gcp.InternalErrorf("unable to find relative path for %q", dir))
		}

		buildables = append(buildables, "./"+rel)
	}

	return buildables
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

func printTipsAndKeepStderrTail(ctx *gcp.Context) gcp.ErrorSummaryProducer {
	return func(result *gcp.ExecResult) *gcp.Error {
		if result.ExitCode != 0 {
			// If `go build` fails with any of those two errors, there's a great chance
			// that we are not building the right package.
			if strings.Contains(result.Stderr, noGoFileError) || strings.Contains(result.Stderr, cannotFindModuleError) {
				ctx.Tipf("Tip: %q env var configures which Go package is built. Default is '.'", env.Buildable)
			}
		}

		return gcp.UserErrorKeepStderrTail(result)
	}
}
