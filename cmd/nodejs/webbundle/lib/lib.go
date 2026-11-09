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

// Package lib implements the google.nodejs.webbundle buildpack for Cloud Run
// framework-aware builds.
//
// The buildpack is named for the bundle it consumes, not for a framework. A
// framework adapter writes .cloudrun/bundle.yaml during the app build to say
// what to keep, what to copy and how to start the app, and BuildFn copies,
// prunes and sets the start command from that file. Another framework can reuse
// this buildpack by having its adapter write the same file. Next.js is the only
// framework with an adapter today (run by google.nodejs.webnextjs), so DetectFn,
// the start-script stand-down rules and the standalone check still assume
// Next.js.
package lib

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/webbundle"
)

const (
	// LabelWebCacheGuard is the label key (prefixed with "google." by
	// ctx.AddLabel) for CDN cache guard telemetry.
	LabelWebCacheGuard = "web-cache-guard"
	// LabelWebServeMode is the label key (prefixed with "google." by
	// ctx.AddLabel) for web serving mode.
	LabelWebServeMode = "web-serve-mode"

	// ServeModeStandalone is the label value when serving from the pruned
	// standalone bundle.
	ServeModeStandalone = "standalone"
	// ServeModeProject is the label value when serving from the unpruned
	// project directory.
	ServeModeProject = "project"
)

var nextConfigFiles = []string{
	"next.config.js",
	"next.config.mjs",
	"next.config.cjs",
	"next.config.ts",
	"next.config.mts",
}

// DetectFn opts in when GOOGLE_WEB_FRAMEWORKS is enabled on the ALPHA release
// track on GCP (!env.IsFAH()), the build isn't a dev mode or dev sync build,
// and the target application is a Next.js app.
//
// TODO(b/578288771): When another framework gets an adapter that writes
// .cloudrun/bundle.yaml, detect that framework's apps here too, and run the
// Next.js-specific checks only for Next.js bundles (metadata.framework).
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !nodejs.WebFrameworksEnabled() {
		return gcp.OptOut(nodejs.WebFrameworksDisabledReason), nil
	}

	appDir := util.ApplicationDirectory(ctx)
	for _, cfg := range nextConfigFiles {
		exists, err := ctx.FileExists(filepath.Join(appDir, cfg))
		if err != nil {
			return nil, err
		}
		if exists {
			return gcp.OptInFileFound(cfg), nil
		}
	}

	pkgJSON, err := nodejs.ReadPackageJSONIfExists(appDir)
	if err != nil {
		return nil, err
	}
	if hasNextDependency(pkgJSON) {
		return gcp.OptIn("next dependency found in package.json"), nil
	}

	if filepath.Clean(appDir) != filepath.Clean(ctx.ApplicationRoot()) {
		nxExists, _ := ctx.FileExists(filepath.Join(ctx.ApplicationRoot(), "nx.json"))
		if pkgJSON == nil || nxExists {
			rootPkgJSON, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
			if err != nil {
				return nil, err
			}
			if hasNextDependency(rootPkgJSON) {
				return gcp.OptIn("next dependency found in root package.json"), nil
			}
		}
	}

	return gcp.OptOut("nextjs application not detected"), nil
}

func hasNextDependency(pkgJSON *nodejs.PackageJSON) bool {
	if pkgJSON == nil {
		return false
	}
	return pkgJSON.Dependencies["next"] != "" || pkgJSON.DevDependencies["next"] != ""
}

// BuildFn validates .cloudrun/bundle.yaml v2, copies static/public assets when
// pruning is enabled, prunes the workspace according to GOOGLE_WEB_PRUNE, and
// sets the CNB web process and labels. Every failure logs a warning via
// ctx.Warnf and falls back cleanly without failing the build.
func BuildFn(ctx *gcp.Context) error {
	rootDir := ctx.ApplicationRoot()
	manifestAbs := filepath.Join(rootDir, filepath.FromSlash(webbundle.DefaultManifestRelPath))
	if _, err := os.Lstat(manifestAbs); os.IsNotExist(err) {
		ctx.Logf("[webbundle] No %s found; skipping web bundle processing.", webbundle.DefaultManifestRelPath)
		return nil
	}

	// STEP 1: Check (.cloudrun/bundle.yaml v2, stand-down rules, runtime
	// manifest, standalone integrity).
	schema, err := webbundle.ReadAndValidate(rootDir)
	if errors.Is(err, webbundle.ErrExportMode) {
		ctx.Warnf("[webbundle] Static export detected (outputMode: export in %s). Cloud Run requires a server process; skipping standalone bundle and leaving default start command unchanged.", webbundle.DefaultManifestRelPath)
		return nil
	}
	if err != nil {
		ctx.Warnf("[webbundle] Invalid %s (%v); skipping web bundle processing and leaving default start command unchanged.", webbundle.DefaultManifestRelPath, err)
		return nil
	}

	appDir := util.ApplicationDirectory(ctx)
	standDownReason, err := nodejs.WebStandDownReason(ctx, appDir)
	if err != nil {
		ctx.Warnf("[webbundle] Failed to inspect start configuration (%v); standing down and leaving default start command unchanged.", err)
		return nil
	}
	if standDownReason != "" {
		ctx.Warnf("[webbundle] Standing down (%s); leaving user start command unchanged.", standDownReason)
		return nil
	}

	if err := webbundle.CheckRuntimeManifest(rootDir, schema); err != nil {
		ctx.Warnf("[webbundle] Corrupt runtime manifest (%v); leaving default start command unchanged.", err)
		return nil
	}

	if err := verifyServeScriptIfUsed(rootDir, schema); err != nil {
		ctx.Warnf("[webbundle] Missing or invalid .cloudrun/serve.js (%v); leaving default start command unchanged.", err)
		return nil
	}

	if err := webbundle.CheckStandalone(rootDir, schema); err != nil {
		ctx.Warnf("[webbundle] Standalone integrity check failed (%v); skipping pruning.", err)
		fallbackToProjectModeIfValid(ctx, rootDir, schema)
		return nil
	}

	rawPruneMode := os.Getenv(webbundle.EnvWebPrune)
	if !webbundle.IsValidPruneMode(rawPruneMode) {
		ctx.Warnf("[webbundle] Unrecognized %s=%q; defaulting to %q.", webbundle.EnvWebPrune, rawPruneMode, webbundle.PruneModeReportOnly)
	}
	pruneMode := webbundle.NormalizePruneMode(rawPruneMode)

	if err := webbundle.CheckBootPackageJSON(rootDir, schema, pruneMode); err != nil {
		ctx.Warnf("[webbundle] Missing boot package.json (%v); leaving default start command unchanged.", err)
		return nil
	}

	// STEP 2: Copy (only when GOOGLE_WEB_PRUNE=true).
	if pruneMode == webbundle.PruneModeTrue {
		if err := webbundle.MergeCopy(rootDir, schema.OutputFiles.ServerApp.Copy); err != nil {
			ctx.Warnf("[webbundle] Failed to copy static/public assets into standalone directory (%v); skipping pruning.", err)
			fallbackToProjectModeIfValid(ctx, rootDir, schema)
			return nil
		}
	}

	// STEP 3: Prune.
	projectModeErr := checkProjectMode(rootDir, schema)
	if pruneMode != webbundle.PruneModeTrue && projectModeErr != nil && userStartUsesStandalone(appDir) {
		ctx.Warnf("[webbundle] Unpruned projectRunCommand is unavailable (%v) while package.json start script uses standalone server.js; preserving standalone directory and leaving start command unchanged.", projectModeErr)
		return nil
	}

	stats, err := webbundle.Prune(rootDir, schema, pruneMode)
	if err != nil {
		ctx.Warnf("[webbundle] Image pruning failed in mode %s (%v); falling back.", pruneMode, err)
		fallbackToProjectModeIfValid(ctx, rootDir, schema)
		return nil
	}
	logPruneStats(ctx, stats)

	// STEP 4: Start command & image labels (CNB web process; last writer wins).
	switch pruneMode {
	case webbundle.PruneModeTrue:
		applyStartCommandAndLabels(ctx, schema.RunConfig.RunCommand, ServeModeStandalone)
	case webbundle.PruneModeReportOnly, webbundle.PruneModeFalse:
		if schema.RunConfig.ProjectRunCommand != "" {
			if err := webbundle.CheckProjectFallbackFiles(rootDir, schema); err != nil {
				ctx.Warnf("[webbundle] Project mode requirements not met (%v); leaving default start command unchanged.", err)
				return nil
			}
			applyStartCommandAndLabels(ctx, schema.RunConfig.ProjectRunCommand, ServeModeProject)
		}
	}
	return nil
}

func userStartUsesStandalone(appDir string) bool {
	pkgJSON, err := nodejs.ReadPackageJSONIfExists(appDir)
	if err != nil || pkgJSON == nil || pkgJSON.Scripts == nil {
		return false
	}
	start := strings.TrimSpace(pkgJSON.Scripts["start"])
	return strings.Contains(start, "standalone/") && strings.HasSuffix(start, "server.js")
}

// checkProjectMode returns why the unpruned project-mode start command can't
// be used, or nil if it can.
func checkProjectMode(rootDir string, schema *webbundle.BundleSchema) error {
	if schema.RunConfig.ProjectRunCommand == "" {
		return errors.New("runConfig.projectRunCommand is not set")
	}
	return webbundle.CheckProjectFallbackFiles(rootDir, schema)
}

func verifyServeScriptIfUsed(rootDir string, schema *webbundle.BundleSchema) error {
	if !webbundle.UsesServeJS(schema.RunConfig.RunCommand) && !webbundle.UsesServeJS(schema.RunConfig.ProjectRunCommand) {
		return nil
	}
	if _, err := webbundle.CleanAndValidatePath(rootDir, ".cloudrun/serve.js", false); err != nil {
		return err
	}
	serveAbs := filepath.Join(rootDir, ".cloudrun", "serve.js")
	info, err := os.Stat(serveAbs)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New(".cloudrun/serve.js is not a regular file")
	}
	return nil
}

func fallbackToProjectModeIfValid(ctx *gcp.Context, rootDir string, schema *webbundle.BundleSchema) {
	if schema.RunConfig.ProjectRunCommand == "" {
		return
	}
	if err := webbundle.CheckProjectFallbackFiles(rootDir, schema); err != nil {
		ctx.Warnf("[webbundle] Fallback project mode requirements not met (%v); leaving default start command unchanged.", err)
		return
	}
	_, _ = webbundle.Prune(rootDir, schema, webbundle.PruneModeFalse)
	applyStartCommandAndLabels(ctx, schema.RunConfig.ProjectRunCommand, ServeModeProject)
}

func applyStartCommandAndLabels(ctx *gcp.Context, cmdStr, serveMode string) {
	parsedCmd, err := webbundle.ParseCommand(cmdStr)
	if err != nil {
		ctx.Warnf("[webbundle] Invalid run command %q (%v); leaving default start command unchanged.", cmdStr, err)
		return
	}
	ctx.AddWebProcess(parsedCmd)
	if webbundle.UsesServeJS(cmdStr) {
		ctx.AddLabel(LabelWebCacheGuard, "true")
		ctx.AddLabel(LabelWebServeMode, serveMode)
	}
}

func logPruneStats(ctx *gcp.Context, stats *webbundle.PruneStats) {
	if stats == nil {
		return
	}
	switch stats.Mode {
	case webbundle.PruneModeTrue:
		ctx.Logf("[webbundle] Pruned workspace (%s=true): removed %d files (%d bytes), kept %d files (%d bytes).", webbundle.EnvWebPrune, stats.RemovedFiles, stats.RemovedBytes, stats.KeptFiles, stats.KeptBytes)
	case webbundle.PruneModeReportOnly:
		ctx.Logf("[webbundle] Prune report-only (%s=report-only): would remove %d files (%d bytes) and keep %d files (%d bytes); removed standalone directory.", webbundle.EnvWebPrune, stats.RemovedFiles, stats.RemovedBytes, stats.KeptFiles, stats.KeptBytes)
		if len(stats.UntracedDataFiles) > 0 {
			ctx.Logf("[webbundle] Untraced data-like files outside standalone trace (%d): %s. If read at runtime via fs, add outputFileTracingIncludes in next.config before setting %s=true.", len(stats.UntracedDataFiles), strings.Join(stats.UntracedDataFiles, ", "), webbundle.EnvWebPrune)
		}
	case webbundle.PruneModeFalse:
		ctx.Logf("[webbundle] Pruning disabled (%s=false); removed standalone directory.", webbundle.EnvWebPrune)
	}
}
