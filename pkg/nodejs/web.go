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

package nodejs

// Web framework (go/framework-aware-cloud-run) helpers shared by the
// google.nodejs.web* buildpacks. Everything here is inert unless
// WebFrameworksEnabled reports true.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"gopkg.in/yaml.v2"
)

const (
	// EnvWebFrameworks enables Cloud Run framework-aware web builds when set to
	// true or 1.
	EnvWebFrameworks = "GOOGLE_WEB_FRAMEWORKS"
	// EnvWebAdapterActive is set by webnextjs during the build when the Next.js
	// adapter is active.
	EnvWebAdapterActive = "GOOGLE_WEB_ADAPTER_ACTIVE"
	// EnvNextJSAdapterVersion overrides the @google-cloud/adapter-nextjs
	// version or tarball path.
	EnvNextJSAdapterVersion = "GOOGLE_NEXTJS_ADAPTER_VERSION"

	// WebFrameworksDisabledReason is the detect opt-out reason that the web*
	// buildpacks report when WebFrameworksEnabled returns false.
	WebFrameworksDisabledReason = "GOOGLE_WEB_FRAMEWORKS is not enabled, the build is not on the ALPHA release track, the target platform is not GCP, or the build uses dev mode or dev sync"
)

// WebFrameworksEnabled reports whether GOOGLE_WEB_FRAMEWORKS is enabled
// (true/1, case-insensitive), the build is on the ALPHA release track
// (env.IsAlphaSupported()), and the build is targeting a standard GCP / Cloud
// Run environment (env.IsGCP(), which excludes App Hosting). It reports false
// for dev mode (GOOGLE_DEVMODE) and dev sync (GOOGLE_DEVSYNC) builds, because
// the web* buildpacks would replace their live-reload web process and prune
// the source files they watch.
func WebFrameworksEnabled() bool {
	if !env.IsGCP() || !env.IsAlphaSupported() {
		return false
	}
	if devMode, _ := env.IsDevMode(); devMode {
		return false
	}
	if devSync, _ := env.IsDevSync(); devSync {
		return false
	}
	v := strings.TrimSpace(os.Getenv(EnvWebFrameworks))
	return strings.EqualFold(v, "true") || v == "1"
}

// WebAdapterActive reports whether WebFrameworksEnabled reports true and
// GOOGLE_WEB_ADAPTER_ACTIVE is enabled (true/1, case-insensitive).
func WebAdapterActive() bool {
	if !WebFrameworksEnabled() {
		return false
	}
	v := strings.TrimSpace(os.Getenv(EnvWebAdapterActive))
	return strings.EqualFold(v, "true") || v == "1"
}

// DecodeYAML unmarshals YAML data into out. It exists because the repository's
// open-source export supports only gopkg.in/yaml.v2 (go.mod pins it, and
// copy.bara.sky maps only third_party/golang/yaml), and pkg/nodejs is already
// an allowlisted user.
// TODO: b/578288771 - Use yamlv3 directly once the export maps third_party/golang/yamlv3.
func DecodeYAML(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}

type pnpmImporterDep struct {
	Version string
}

func (d *pnpmImporterDep) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err == nil { // if NO error
		d.Version = s
		return nil
	}
	var m struct {
		Version string `yaml:"version"`
	}
	if err := unmarshal(&m); err != nil {
		return err
	}
	d.Version = m.Version
	return nil
}

type pnpmImporter struct {
	Dependencies    map[string]pnpmImporterDep `yaml:"dependencies"`
	DevDependencies map[string]pnpmImporterDep `yaml:"devDependencies"`
}

type pnpmWorkspaceLockfile struct {
	Importers       map[string]pnpmImporter    `yaml:"importers"`
	Dependencies    map[string]pnpmImporterDep `yaml:"dependencies"`
	DevDependencies map[string]pnpmImporterDep `yaml:"devDependencies"`
}

// WorkspaceLockfileVersion returns the resolved version of pkg for a monorepo
// workspace at appDir from a pnpm or npm lockfile in appDir or rootDir. It
// returns ("", nil) when appDir equals rootDir, when no pnpm or npm lockfile is
// present, or when pkg is not found in the lockfile.
func WorkspaceLockfileVersion(rootDir, appDir, pkg string) (string, error) {
	if rootDir == "" || appDir == "" || pkg == "" {
		return "", nil
	}
	cleanRoot := filepath.Clean(rootDir)
	cleanApp := filepath.Clean(appDir)
	if cleanRoot == cleanApp {
		return "", nil
	}
	rel, err := filepath.Rel(cleanRoot, cleanApp)
	if err != nil {
		return "", nil
	}
	relSlash := filepath.ToSlash(filepath.Clean(rel))
	if relSlash == "" || relSlash == "." || strings.HasPrefix(relSlash, "..") {
		return "", nil
	}

	for _, dir := range []string{cleanApp, cleanRoot} {
		pnpmPath := filepath.Join(dir, PNPMLock)
		if ver, err := workspacePnpmLockVersion(pnpmPath, relSlash, pkg); err != nil || ver != "" {
			return ver, err
		}
		for _, npmFile := range []string{NPMShrinkwrap, PackageLock} {
			npmPath := filepath.Join(dir, npmFile)
			if ver, err := workspaceNPMLockVersion(npmPath, relSlash, pkg); err != nil || ver != "" {
				return ver, err
			}
		}
	}
	return "", nil
}

func workspacePnpmLockVersion(lockPath, relSlash, pkg string) (string, error) {
	raw, err := os.ReadFile(lockPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", gcp.UserErrorf("reading %s: %w", lockPath, err)
	}
	var lockfile pnpmWorkspaceLockfile
	if err := yaml.Unmarshal(raw, &lockfile); err != nil {
		return "", gcp.UserErrorf("parsing %s: %w", lockPath, err)
	}
	for _, key := range []string{relSlash, "."} {
		imp, ok := lockfile.Importers[key]
		if !ok {
			continue
		}
		if dep, ok := imp.Dependencies[pkg]; ok && dep.Version != "" {
			return cleanPnpmVersion(dep.Version), nil
		}
		if dep, ok := imp.DevDependencies[pkg]; ok && dep.Version != "" {
			return cleanPnpmVersion(dep.Version), nil
		}
	}
	if dep, ok := lockfile.Dependencies[pkg]; ok && dep.Version != "" {
		return cleanPnpmVersion(dep.Version), nil
	}
	if dep, ok := lockfile.DevDependencies[pkg]; ok && dep.Version != "" {
		return cleanPnpmVersion(dep.Version), nil
	}
	return "", nil
}

func cleanPnpmVersion(ver string) string {
	return strings.TrimSpace(strings.Split(ver, "(")[0])
}

func workspaceNPMLockVersion(lockPath, relSlash, pkg string) (string, error) {
	raw, err := os.ReadFile(lockPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", gcp.UserErrorf("reading %s: %w", lockPath, err)
	}
	var lockfile NPMLockfile
	if err := json.Unmarshal(raw, &lockfile); err != nil {
		return "", gcp.UserErrorf("parsing %s: %w", lockPath, err)
	}
	for _, key := range []string{relSlash + "/node_modules/" + pkg, "node_modules/" + pkg} {
		if entry, ok := lockfile.Packages[key]; ok && strings.TrimSpace(entry.Version) != "" {
			return strings.TrimSpace(entry.Version), nil
		}
	}
	return "", nil
}

// BunLockVersion returns the resolved version of pkg from a text bun.lock in
// dir ("" if absent). nodejs.Version remains unchanged so CheckVulnerabilities
// behavior for Bun is unaffected.
// TODO: b/582363483 - Add bun.lock support to nodejs.Version() and remove this helper.
func BunLockVersion(dir, pkg string) (string, error) {
	if dir == "" || pkg == "" {
		return "", nil
	}
	lockPath := dir
	if info, err := os.Stat(dir); err == nil && info.IsDir() { // if NO error
		lockPath = filepath.Join(dir, BunLock)
	} else if !strings.HasSuffix(dir, BunLock) {
		lockPath = filepath.Join(dir, BunLock)
	}
	raw, err := os.ReadFile(lockPath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", gcp.UserErrorf("reading %s: %w", lockPath, err)
	}
	cleaned := stripJSONC(raw)
	var parsed struct {
		Packages map[string]json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(cleaned, &parsed); err != nil {
		return "", gcp.UserErrorf("parsing %s: %w", lockPath, err)
	}
	if len(parsed.Packages) == 0 {
		return "", nil
	}
	if ver := extractBunPackageEntryVersion(parsed.Packages[pkg], pkg); ver != "" {
		return ver, nil
	}
	suffix := "/" + pkg
	for k, entry := range parsed.Packages {
		if strings.HasSuffix(k, suffix) {
			if ver := extractBunPackageEntryVersion(entry, pkg); ver != "" {
				return ver, nil
			}
		}
	}
	return "", nil
}

func extractBunPackageEntryVersion(raw json.RawMessage, pkg string) string {
	if len(raw) == 0 {
		return ""
	}
	var tuple []json.RawMessage
	if err := json.Unmarshal(raw, &tuple); err == nil && len(tuple) > 0 { // if NO error
		var ident string
		if err := json.Unmarshal(tuple[0], &ident); err == nil && ident != "" { // if NO error
			prefix := pkg + "@"
			if strings.HasPrefix(ident, prefix) {
				ver := strings.TrimPrefix(ident, prefix)
				ver = strings.TrimPrefix(ver, "npm:")
				return ver
			}
			if !strings.Contains(ident, "@") {
				return ident
			}
		}
	}
	var obj struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Version != "" { // if NO error
		return obj.Version
	}
	return ""
}

// stripJSONC strips line/block comments and trailing commas from JSONC input
// (such as bun.lock).
func stripJSONC(src []byte) []byte {
	noComments := make([]byte, 0, len(src))
	inString := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			noComments = append(noComments, c)
			if c == '\\' && i+1 < len(src) {
				i++
				noComments = append(noComments, src[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			noComments = append(noComments, c)
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			if i < len(src) {
				noComments = append(noComments, '\n')
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++ // skip '/'
			continue
		}
		noComments = append(noComments, c)
	}

	out := make([]byte, 0, len(noComments))
	inString = false
	for i := 0; i < len(noComments); i++ {
		c := noComments[i]
		if inString {
			out = append(out, c)
			if c == '\\' && i+1 < len(noComments) {
				i++
				out = append(out, noComments[i])
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(noComments) && (noComments[j] == ' ' || noComments[j] == '\t' || noComments[j] == '\r' || noComments[j] == '\n') {
				j++
			}
			if j < len(noComments) && (noComments[j] == '}' || noComments[j] == ']') {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// WebStandDownReason returns "" when the Next.js app in appDir can be served by
// .cloudrun/serve.js, or a short human-readable reason (used verbatim in a
// build warning) when the web path must stand down: custom server / explicit
// start, unsupported `next start` flags or env prefixes, no start script but
// server.js or main, Procfile, GOOGLE_ENTRYPOINT, or (GOOGLE_BUILDABLE set) a
// root start script that is not absent / standard / an app-only runner. appDir
// is absolute.
func WebStandDownReason(ctx *gcp.Context, appDir string) (string, error) {
	rootDir := ctx.ApplicationRoot()
	buildable := strings.TrimSpace(os.Getenv(env.Buildable))
	if appDir == "" {
		appDir = rootDir
		if buildable != "" && rootDir != "" {
			appDir = filepath.Join(rootDir, buildable)
		}
	}

	if ep := strings.TrimSpace(os.Getenv(env.Entrypoint)); ep != "" {
		return fmt.Sprintf("GOOGLE_ENTRYPOINT is set (%q)", ep), nil
	}

	procfileDirs := []string{rootDir}
	if appDir != "" && filepath.Clean(appDir) != filepath.Clean(rootDir) {
		procfileDirs = append(procfileDirs, appDir)
	}
	for _, dir := range procfileDirs {
		if dir == "" {
			continue
		}
		hasProcfile, err := ctx.FileExists(dir, "Procfile")
		if err != nil {
			return "", err
		}
		if hasProcfile {
			return "Procfile is present", nil
		}
	}

	appPJS, err := ReadPackageJSONIfExists(appDir)
	if err != nil {
		return "", err
	}

	var startScript string
	if appPJS != nil && appPJS.Scripts != nil {
		startScript = strings.TrimSpace(appPJS.Scripts["start"])
	}

	if startScript != "" {
		if ok, reason := validateNextStartScript(startScript); !ok {
			return reason, nil
		}
	} else {
		if appPJS != nil && strings.TrimSpace(appPJS.Main) != "" {
			return fmt.Sprintf(`package.json has no "start" script and sets "main" to %q`, appPJS.Main), nil
		}
		if appDir != "" {
			hasServerJS, err := ctx.FileExists(appDir, "server.js")
			if err != nil {
				return "", err
			}
			if hasServerJS {
				return `package.json has no "start" script and server.js is present`, nil
			}
		}
	}

	if buildable != "" && rootDir != "" && filepath.Clean(appDir) != filepath.Clean(rootDir) {
		rootPJS, err := ReadPackageJSONIfExists(rootDir)
		if err != nil {
			return "", err
		}
		if rootPJS != nil && rootPJS.Scripts != nil {
			rootStart := strings.TrimSpace(rootPJS.Scripts["start"])
			if rootStart != "" {
				if ok, _ := validateNextStartScript(rootStart); !ok && !isMonorepoAppRunnerStart(rootStart, buildable, appDir, appPJS) {
					return fmt.Sprintf(`root package.json "start" script %q is a custom start command not scoped to %q`, rootStart, filepath.ToSlash(filepath.Clean(buildable))), nil
				}
			}
		}
	}

	return "", nil
}

func validateNextStartScript(script string) (bool, string) {
	if strings.ContainsAny(script, ";&|><`") || strings.Contains(script, "$(") {
		return false, fmt.Sprintf(`package.json "start" script %q is a custom start command`, script)
	}
	tokens := strings.Fields(script)
	if len(tokens) == 0 {
		return true, ""
	}
	first := tokens[0]
	if strings.Contains(first, "=") || first == "cross-env" || first == "env" {
		return false, fmt.Sprintf(`package.json "start" script %q uses an environment prefix`, script)
	}

	if first == "next" || first == "node_modules/.bin/next" || first == "./node_modules/.bin/next" {
		if len(tokens) < 2 || tokens[1] != "start" {
			return false, fmt.Sprintf(`package.json "start" script %q is a custom start command`, script)
		}
		for i := 2; i < len(tokens); i++ {
			arg := tokens[i]
			switch {
			case arg == "-p" || arg == "--port" || arg == "-H" || arg == "--hostname":
				if i+1 >= len(tokens) || strings.HasPrefix(tokens[i+1], "-") {
					return false, fmt.Sprintf(`package.json "start" script %q has incomplete flag %q`, script, arg)
				}
				i++
			case strings.HasPrefix(arg, "--port=") && len(arg) > len("--port="):
			case strings.HasPrefix(arg, "-p=") && len(arg) > len("-p="):
			case strings.HasPrefix(arg, "--hostname=") && len(arg) > len("--hostname="):
			case strings.HasPrefix(arg, "-H=") && len(arg) > len("-H="):
			default:
				return false, fmt.Sprintf(`package.json "start" script %q passes unsupported flag or argument %q to next start`, script, arg)
			}
		}
		return true, ""
	}

	if first == "node" && len(tokens) == 2 {
		cleanTarget := filepath.ToSlash(filepath.Clean(tokens[1]))
		if !strings.HasPrefix(cleanTarget, "/") && !strings.HasPrefix(cleanTarget, "../") && cleanTarget != ".." && !strings.Contains(cleanTarget, "/../") {
			if cleanTarget == "standalone/server.js" || strings.HasSuffix(cleanTarget, "/standalone/server.js") || (strings.Contains(cleanTarget, "/standalone/") && strings.HasSuffix(cleanTarget, "/server.js")) {
				return true, ""
			}
		}
		return false, fmt.Sprintf(`package.json "start" script %q runs a custom server`, script)
	}

	return false, fmt.Sprintf(`package.json "start" script %q is a custom start command`, script)
}

func isMonorepoAppRunnerStart(script, buildable, appDir string, appPJS *PackageJSON) bool {
	if strings.ContainsAny(script, ";&|><`") || strings.Contains(script, "$(") {
		return false
	}
	tokens := strings.Fields(script)
	if len(tokens) == 0 {
		return false
	}
	cleanDir := filepath.ToSlash(filepath.Clean(buildable))
	dotCleanDir := "./" + cleanDir
	var appName string
	if appPJS != nil {
		appName = strings.TrimSpace(appPJS.Name)
	}
	var nxName string
	if nxProj, err := ReadNxProjectJSONIfExists(appDir); err == nil && nxProj != nil { // if NO error
		nxName = strings.TrimSpace(nxProj.Name)
	}

	matchTarget := func(val string, allowed ...string) bool {
		val = strings.Trim(val, `"'`)
		if val == "" {
			return false
		}
		for _, a := range allowed {
			if a != "" && val == a {
				return true
			}
		}
		return false
	}

	switch tokens[0] {
	case "turbo":
		// turbo run start --filter=<app> | turbo [run] start --filter <app>
		idx := 1
		if idx < len(tokens) && tokens[idx] == "run" {
			idx++
		}
		if idx >= len(tokens) || tokens[idx] != "start" {
			return false
		}
		idx++
		if idx == len(tokens)-1 && strings.HasPrefix(tokens[idx], "--filter=") {
			return matchTarget(strings.TrimPrefix(tokens[idx], "--filter="), appName, cleanDir, dotCleanDir)
		}
		if idx == len(tokens)-2 && tokens[idx] == "--filter" {
			return matchTarget(tokens[idx+1], appName, cleanDir, dotCleanDir)
		}
		return false

	case "nx":
		// nx run <app>:start | nx start <app>
		if len(tokens) == 3 && tokens[1] == "run" && strings.HasSuffix(tokens[2], ":start") {
			target := strings.TrimSuffix(tokens[2], ":start")
			return matchTarget(target, nxName, appName, cleanDir)
		}
		if len(tokens) == 3 && tokens[1] == "start" {
			return matchTarget(tokens[2], nxName, appName, cleanDir)
		}
		return false

	case "npm":
		// npm run start -w <dir> | npm run start --workspace=<dir>
		// npm -w <dir> run start
		if len(tokens) == 5 && tokens[1] == "run" && tokens[2] == "start" && (tokens[3] == "-w" || tokens[3] == "--workspace") {
			return matchTarget(tokens[4], cleanDir, dotCleanDir, appName)
		}
		if len(tokens) == 4 && tokens[1] == "run" && tokens[2] == "start" && strings.HasPrefix(tokens[3], "--workspace=") {
			return matchTarget(strings.TrimPrefix(tokens[3], "--workspace="), cleanDir, dotCleanDir, appName)
		}
		if len(tokens) == 5 && (tokens[1] == "-w" || tokens[1] == "--workspace") && tokens[3] == "run" && tokens[4] == "start" {
			return matchTarget(tokens[2], cleanDir, dotCleanDir, appName)
		}
		if len(tokens) == 4 && strings.HasPrefix(tokens[1], "--workspace=") && tokens[2] == "run" && tokens[3] == "start" {
			return matchTarget(strings.TrimPrefix(tokens[1], "--workspace="), cleanDir, dotCleanDir, appName)
		}
		return false

	case "pnpm":
		// pnpm --filter <app> [run] start | pnpm --filter=<app> [run] start
		if len(tokens) >= 3 && (tokens[1] == "--filter" || tokens[1] == "-F") {
			rest := tokens[3:]
			if (len(rest) == 1 && rest[0] == "start") || (len(rest) == 2 && rest[0] == "run" && rest[1] == "start") {
				return matchTarget(tokens[2], appName, cleanDir, dotCleanDir)
			}
		}
		if len(tokens) >= 3 && strings.HasPrefix(tokens[1], "--filter=") {
			rest := tokens[2:]
			if (len(rest) == 1 && rest[0] == "start") || (len(rest) == 2 && rest[0] == "run" && rest[1] == "start") {
				return matchTarget(strings.TrimPrefix(tokens[1], "--filter="), appName, cleanDir, dotCleanDir)
			}
		}
		return false

	case "yarn":
		// yarn workspace <app> start | yarn workspace <app> run start
		if len(tokens) >= 4 && tokens[1] == "workspace" {
			rest := tokens[3:]
			if (len(rest) == 1 && rest[0] == "start") || (len(rest) == 2 && rest[0] == "run" && rest[1] == "start") {
				return matchTarget(tokens[2], appName, cleanDir)
			}
		}
		return false
	}

	return false
}
