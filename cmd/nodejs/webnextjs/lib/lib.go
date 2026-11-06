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

// Package lib implements the google.nodejs.webnextjs buildpack for Cloud Run.
package lib

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/ar"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/util"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/nodejs"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

const (
	adapterLayerName      = "web_nextjs_adapter"
	adapterPackageName    = "@google-cloud/adapter-nextjs"
	defaultAdapterVersion = "0.1.0"
	// supportedNextVersions is the Next.js range that the adapter supports,
	// >=16.3.5 <17.0.0. The "-0" upper bound makes the semver check accept 16.x
	// prereleases such as 16.5.0-canary.1, as the adapter's own version check
	// does. 16.3.5 and 17.0.0 prereleases stay outside the range.
	supportedNextVersions = ">=16.3.5 <17.0.0-0"
)

var (
	// supportedNextConfigFiles lists all Next.js config filenames checked
	// during detection.
	supportedNextConfigFiles = []string{
		"next.config.js",
		"next.config.mjs",
		"next.config.cjs",
		"next.config.ts",
		"next.config.mts",
	}
	semverTokenRegex = regexp.MustCompile(`(\d+)(?:\.(\d+|x|X|\*))?(?:\.(\d+|x|X|\*))?(?:-([0-9A-Za-z.-]+))?`)
)

// DetectFn detects whether the application is a Next.js project eligible for
// Cloud Run framework-aware builds.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if !nodejs.WebFrameworksEnabled() {
		return gcp.OptOut(nodejs.WebFrameworksDisabledReason), nil
	}

	appDir := util.ApplicationDirectory(ctx)
	for _, cfg := range supportedNextConfigFiles {
		exists, err := ctx.FileExists(appDir, cfg)
		if err != nil {
			return nil, err
		}
		if exists {
			return gcp.OptInFileFound(cfg), nil
		}
	}

	appPJS, err := nodejs.ReadPackageJSONIfExists(appDir)
	if err != nil {
		return nil, err
	}
	if hasPackageDependency(appPJS, "next") {
		return gcp.OptIn("next dependency found in package.json"), nil
	}

	if appDir != ctx.ApplicationRoot() {
		nxExists, err := ctx.FileExists(ctx.ApplicationRoot(), "nx.json")
		if err != nil {
			return nil, err
		}
		if appPJS == nil || nxExists {
			rootPJS, err := nodejs.ReadPackageJSONIfExists(ctx.ApplicationRoot())
			if err != nil {
				return nil, err
			}
			if hasPackageDependency(rootPJS, "next") {
				return gcp.OptIn("next dependency found in root package.json"), nil
			}
		}
	}

	return gcp.OptOut("next.config.* and next dependency not found"), nil
}

// BuildFn validates stand-down conditions, installs
// @google-cloud/adapter-nextjs in a build-only layer, and configures
// build-time environment variables for Next.js adapter integration.
func BuildFn(ctx *gcp.Context) error {
	appDir := util.ApplicationDirectory(ctx)
	rootDir := ctx.ApplicationRoot()

	// Remove any stale or accidentally committed /workspace/.cloudrun
	// directory before build.
	if err := ctx.RemoveAll(filepath.Join(rootDir, ".cloudrun")); err != nil {
		return err
	}

	// Stand-down 1: GOOGLE_FUNCTION_TARGET set.
	if strings.TrimSpace(os.Getenv(env.FunctionTarget)) != "" {
		ctx.Warnf("[webnextjs] Standing down: %s is set; deferring to Functions Framework buildpacks.", env.FunctionTarget)
		return nil
	}

	appPJS, err := nodejs.ReadPackageJSONIfExists(appDir)
	if err != nil {
		return err
	}
	var rootPJS *nodejs.PackageJSON
	if appDir != rootDir {
		rootPJS, err = nodejs.ReadPackageJSONIfExists(rootDir)
		if err != nil {
			return err
		}
	}

	// Stand-down 2: @apphosting/adapter-nextjs present in dependencies or
	// devDependencies.
	if hasPackageDependency(appPJS, "@apphosting/adapter-nextjs") || hasPackageDependency(rootPJS, "@apphosting/adapter-nextjs") {
		ctx.Warnf("[webnextjs] Standing down: @apphosting/adapter-nextjs is listed in package.json; skipping Cloud Run Next.js adapter to avoid conflicting adapter hooks.")
		return nil
	}

	// Stand-down 3: Next.js version outside >= 16.3.5 and < 17.0.0.
	nextVer := resolveNextVersion(ctx, appDir, rootDir, appPJS, rootPJS)
	if nextVer != "" {
		// A version that doesn't parse, such as a "canary" dist-tag, isn't a
		// stand-down: the adapter checks the installed Next.js version again
		// during `next build`.
		if supported, err := nodejs.VersionMatchesSemver(ctx, supportedNextVersions, nextVer); err == nil && !supported {
			ctx.Warnf("[webnextjs] Standing down: resolved Next.js version %q is outside the supported range (>=16.3.5 <17.0.0); continuing with generic Node.js build.", nextVer)
			return nil
		}
	}

	// Stand-down 4: Custom server / entrypoint / Procfile check.
	standDownReason, err := nodejs.WebStandDownReason(ctx, appDir)
	if err != nil {
		return err
	}
	if standDownReason != "" {
		ctx.Warnf("[webnextjs] Standing down: %s; continuing with generic Node.js build.", standDownReason)
		return nil
	}

	// Install @google-cloud/adapter-nextjs into a build-only layer (never a
	// launch or cache layer).
	adapterLayer, err := ctx.Layer(adapterLayerName, gcp.BuildLayer)
	if err != nil {
		return err
	}

	adapterSpec, recordedAdapterVersion := resolveAdapterSpec()
	if err := ar.GenerateNPMConfig(ctx); err != nil {
		ctx.Warnf("[webnextjs] Standing down: failed to generate Artifact Registry credentials (%v); continuing with generic Node.js build.", err)
		return nil
	}
	if err := copyProjectNPMRC(ctx, rootDir, adapterLayer.Path); err != nil {
		ctx.Warnf("[webnextjs] Standing down: failed to copy .npmrc into the adapter layer (%v); continuing with generic Node.js build.", err)
		return nil
	}
	if _, err := ctx.Exec([]string{"npm", "install", "--prefix", adapterLayer.Path, adapterSpec}, gcp.WithUserAttribution); err != nil {
		ctx.Warnf("[webnextjs] Standing down: failed to install %s (%v); continuing with generic Node.js build.", adapterSpec, err)
		return nil
	}

	configureAdapterLayerEnv(adapterLayer, rootDir)

	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.FrameworkName, "nextjs")
	if nextVer != "" {
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.FrameworkVersion, buildermetadata.MetadataValue(nextVer))
	}
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.AdapterName, adapterPackageName)
	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.AdapterVersion, buildermetadata.MetadataValue(recordedAdapterVersion))

	return nil
}

// copyProjectNPMRC copies <rootDir>/.npmrc, if present, to <layerDir>/.npmrc.
// `npm install --prefix <layerDir>` reads the project config from
// <layerDir>/.npmrc rather than the app's, so without this copy the app's
// registry settings (for example an Artifact Registry repository or a
// corporate mirror) would not apply to the adapter install. Artifact Registry
// tokens stay in the user-level ~/.npmrc that ar.GenerateNPMConfig writes.
func copyProjectNPMRC(ctx *gcp.Context, rootDir, layerDir string) error {
	src := filepath.Join(rootDir, ".npmrc")
	exists, err := ctx.FileExists(src)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	data, err := ctx.ReadFile(src)
	if err != nil {
		return err
	}
	return ctx.WriteFile(filepath.Join(layerDir, ".npmrc"), data, 0644)
}

func configureAdapterLayerEnv(adapterLayer *libcnb.Layer, rootDir string) {
	if adapterLayer.BuildEnvironment == nil {
		adapterLayer.BuildEnvironment = make(libcnb.Environment)
	}
	adapterEntryPath := filepath.Join(adapterLayer.Path, "node_modules", "@google-cloud", "adapter-nextjs")
	adapterLayer.BuildEnvironment.Default("NEXT_ADAPTER_PATH", adapterEntryPath)
	adapterLayer.BuildEnvironment.Default("NEXT_PRIVATE_OUTPUT_TRACE_ROOT", rootDir)
	adapterLayer.BuildEnvironment.Default("NEXT_TELEMETRY_DISABLED", "1")
	adapterLayer.BuildEnvironment.Override(nodejs.EnvWebAdapterActive, "true")
}

func hasPackageDependency(pjs *nodejs.PackageJSON, pkg string) bool {
	return pjs != nil && (pjs.Dependencies[pkg] != "" || pjs.DevDependencies[pkg] != "")
}

func packageDependencySpec(pjs *nodejs.PackageJSON, pkg string) string {
	if pjs == nil {
		return ""
	}
	if v := strings.TrimSpace(pjs.Dependencies[pkg]); v != "" {
		return v
	}
	return strings.TrimSpace(pjs.DevDependencies[pkg])
}

func resolveNextVersion(ctx *gcp.Context, appDir, rootDir string, appPJS, rootPJS *nodejs.PackageJSON) string {
	dirs := []string{appDir}
	if filepath.Clean(appDir) != filepath.Clean(rootDir) {
		dirs = append(dirs, rootDir)
		if ver, err := nodejs.WorkspaceLockfileVersion(rootDir, appDir, "next"); err == nil && ver != "" { // if NO error
			if _, err := semver.NewVersion(ver); err == nil { // if NO error
				return ver
			}
		}
	}

	// 1. Check npm/pnpm/yarn lockfile via ReadNodeDependencies + nodejs.Version.
	for _, dir := range dirs {
		nodeDeps, err := nodejs.ReadNodeDependencies(ctx, dir)
		if err != nil || nodeDeps == nil || nodeDeps.LockfilePath == "" {
			continue
		}
		ver, err := nodejs.Version(nodeDeps, "next")
		if err != nil || ver == "" {
			continue
		}
		if _, err := semver.NewVersion(ver); err == nil { // if NO error
			return ver
		}
	}

	// 2. Check bun.lock via nodejs.BunLockVersion in appDir or rootDir.
	for _, dir := range dirs {
		exists, err := ctx.FileExists(dir, "bun.lock")
		if err != nil || !exists {
			continue
		}
		ver, err := nodejs.BunLockVersion(dir, "next")
		if err != nil || ver == "" {
			continue
		}
		if _, err := semver.NewVersion(ver); err == nil { // if NO error
			return ver
		}
	}

	// 3. Fall back to the lowest version allowed by the package.json range.
	rawSpec := packageDependencySpec(appPJS, "next")
	if rawSpec == "" {
		rawSpec = packageDependencySpec(rootPJS, "next")
	}
	if rawSpec == "" {
		return ""
	}
	return lowestVersionFromRange(rawSpec)
}

// lowestVersionFromRange extracts the lowest semver version from a
// package.json dependency range.
func lowestVersionFromRange(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	// If it already parses directly as an exact version, return it.
	if v, err := semver.NewVersion(spec); err == nil && !strings.ContainsAny(spec, "^~><=|*xX ") { // if NO error
		return v.String()
	}

	var lowest *semver.Version
	for _, orBranch := range strings.Split(spec, "||") {
		orBranch = strings.TrimSpace(orBranch)
		if orBranch == "" {
			continue
		}
		// In a hyphen range "A - B", only consider the lower bound "A".
		if idx := strings.Index(orBranch, " - "); idx != -1 {
			orBranch = orBranch[:idx]
		}
		// Find tokens in this comparator set; ignore upper bounds starting with '<'.
		for _, token := range strings.Fields(orBranch) {
			if strings.HasPrefix(token, "<") {
				continue
			}
			m := semverTokenRegex.FindStringSubmatch(token)
			if m == nil {
				continue
			}
			major := m[1]
			minor := normalizeRangePart(m[2])
			patch := normalizeRangePart(m[3])
			candidateStr := major + "." + minor + "." + patch
			if m[4] != "" {
				candidateStr += "-" + m[4]
			}
			v, err := semver.NewVersion(candidateStr)
			if err != nil {
				continue
			}
			if lowest == nil || v.LessThan(lowest) {
				lowest = v
			}
			break
		}
	}
	if lowest != nil {
		return lowest.String()
	}
	return spec
}

func normalizeRangePart(part string) string {
	if part == "" || part == "x" || part == "X" || part == "*" {
		return "0"
	}
	return part
}

func resolveAdapterSpec() (spec, version string) {
	raw := strings.TrimSpace(os.Getenv(nodejs.EnvNextJSAdapterVersion))
	if raw == "" {
		raw = defaultAdapterVersion
	}
	if isLocalTarballOrFileSpec(raw) || strings.HasPrefix(raw, adapterPackageName+"@") {
		return raw, raw
	}
	return adapterPackageName + "@" + raw, raw
}

func isLocalTarballOrFileSpec(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "file:") ||
		strings.HasSuffix(s, ".tgz") ||
		strings.HasSuffix(s, ".tar.gz")
}
