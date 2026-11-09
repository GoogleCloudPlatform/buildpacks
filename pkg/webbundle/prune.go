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

package webbundle

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// KeepSet represents the symlink-closed set of workspace-relative paths to keep
// during pruning.
type KeepSet struct {
	// KeepAll is true when "." is in the include list.
	KeepAll bool
	// Subtrees holds workspace-relative slash paths whose entire subtree is kept.
	Subtrees map[string]bool
	// Ancestors holds strict parent directories of Subtrees that must be kept
	// and descended into.
	Ancestors map[string]bool
}

// IsKeptSubtree reports whether rel (a cleaned slash-separated
// workspace-relative path) is in Subtrees or under any path in Subtrees.
func (ks *KeepSet) IsKeptSubtree(rel string) bool {
	if ks == nil {
		return false
	}
	if ks.KeepAll {
		return true
	}
	rel = path.Clean(rel)
	if ks.Subtrees[rel] {
		return true
	}
	for cur := path.Dir(rel); cur != "." && cur != "/" && cur != ""; cur = path.Dir(cur) {
		if ks.Subtrees[cur] {
			return true
		}
	}
	return false
}

// IsAncestor reports whether rel is a strict ancestor directory of a kept
// subtree.
func (ks *KeepSet) IsAncestor(rel string) bool {
	if ks == nil {
		return false
	}
	if ks.KeepAll {
		return true
	}
	return ks.Ancestors[path.Clean(rel)]
}

// ComputeKeepSet computes the two-pass symlink-closed keep-set for the given
// workspace and include list. It returns ErrEmptyInclude when include is empty.
func ComputeKeepSet(workspaceRoot string, include []string) (*KeepSet, error) {
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}
	if len(include) == 0 {
		return nil, ErrEmptyInclude
	}

	ks := &KeepSet{
		Subtrees:  make(map[string]bool),
		Ancestors: make(map[string]bool),
	}

	var queue []string
	var enqueueSubtree func(rel string)
	enqueueSubtree = func(rel string) {
		rel = path.Clean(rel)
		if rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
			return
		}
		if rel == "." {
			ks.KeepAll = true
			return
		}
		if ks.Subtrees[rel] {
			return
		}
		ks.Subtrees[rel] = true
		queue = append(queue, rel)

		if pnpmNM := pnpmVirtualStoreNodeModules(rel); pnpmNM != "" && pnpmNM != rel {
			enqueueSubtree(pnpmNM)
		}

		for parent := path.Dir(rel); parent != "." && parent != "/" && parent != ""; parent = path.Dir(parent) {
			ks.Ancestors[parent] = true
			parentAbs := filepath.Join(realRoot, filepath.FromSlash(parent))
			if info, err := os.Lstat(parentAbs); err == nil && info.Mode()&os.ModeSymlink != 0 {
				if !ks.Subtrees[parent] {
					ks.Subtrees[parent] = true
					queue = append(queue, parent)
				}
			}
		}
	}

	for _, rawInc := range include {
		inc, err := CleanAndValidatePath(realRoot, rawInc, true)
		if err != nil {
			return nil, fmt.Errorf("invalid include path %q: %v", rawInc, err)
		}
		enqueueSubtree(inc)
		if ks.KeepAll {
			return ks, nil
		}
	}
	enqueueSubtree(CloudRunDirName)

	visitedWalkDirs := make(map[string]bool)

	enqueueSymlinkTargets := func(symlinkAbs string) {
		cur := symlinkAbs
		for depth := 0; depth < maxSymlinkDepth; depth++ {
			info, err := os.Lstat(cur)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				break
			}
			linkTarget, err := os.Readlink(cur)
			if err != nil {
				break
			}
			var targetAbs string
			if filepath.IsAbs(linkTarget) {
				targetAbs = filepath.Clean(linkTarget)
			} else {
				targetAbs = filepath.Clean(filepath.Join(filepath.Dir(cur), linkTarget))
			}
			if !isWithinRealRoot(realRoot, targetAbs) {
				break
			}
			rel, err := filepath.Rel(realRoot, targetAbs)
			if err != nil {
				break
			}
			enqueueSubtree(filepath.ToSlash(rel))
			if ks.KeepAll {
				return
			}
			cur = targetAbs
		}
		if evalAbs, err := filepath.EvalSymlinks(symlinkAbs); err == nil && isWithinRealRoot(realRoot, evalAbs) {
			if rel, err := filepath.Rel(realRoot, evalAbs); err == nil {
				enqueueSubtree(filepath.ToSlash(rel))
			}
		}
	}

	for len(queue) > 0 && !ks.KeepAll {
		rel := queue[0]
		queue = queue[1:]

		absPath := filepath.Join(realRoot, filepath.FromSlash(rel))
		info, err := os.Lstat(absPath)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			enqueueSymlinkTargets(absPath)
			continue
		}

		if info.IsDir() {
			if visitedWalkDirs[rel] {
				continue
			}
			visitedWalkDirs[rel] = true

			_ = filepath.WalkDir(absPath, func(p string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if !isWithinRealRoot(realRoot, p) {
					if d != nil && d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if d.Type()&os.ModeSymlink != 0 {
					enqueueSymlinkTargets(p)
				}
				return nil
			})
		}
	}

	return ks, nil
}

func pnpmVirtualStoreNodeModules(rel string) string {
	segments := strings.Split(rel, "/")
	for i, seg := range segments {
		if seg == ".pnpm" {
			for j := i + 1; j < len(segments); j++ {
				if segments[j] == "node_modules" {
					return strings.Join(segments[:j+1], "/")
				}
			}
		}
	}
	return ""
}

// PruneStats holds metrics and report-only findings from a Prune run.
type PruneStats struct {
	Mode              string
	RemovedFiles      int
	RemovedBytes      int64
	KeptFiles         int
	KeptBytes         int64
	UntracedDataFiles []string
}

// Prune executes STEP 3 of the webbundle lifecycle according to mode ("true",
// "report-only", or "false").
func Prune(workspaceRoot string, schema *BundleSchema, mode string) (*PruneStats, error) {
	if schema == nil {
		return nil, errors.New("bundle schema is nil")
	}
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}

	normalizedMode := NormalizePruneMode(mode)
	stats := &PruneStats{
		Mode: normalizedMode,
	}

	if normalizedMode == PruneModeFalse {
		if err := removeStandaloneDir(realRoot, schema); err != nil {
			return stats, err
		}
		return stats, nil
	}

	ks, err := ComputeKeepSet(realRoot, schema.OutputFiles.ServerApp.Include)
	if err != nil {
		return nil, err
	}

	if ks.KeepAll {
		files, bytes := countSubtreeStats(realRoot)
		stats.KeptFiles = files
		stats.KeptBytes = bytes
		if normalizedMode == PruneModeReportOnly {
			if err := removeStandaloneDir(realRoot, schema); err != nil {
				return stats, err
			}
		}
		return stats, nil
	}

	untracedSeen := make(map[string]bool)

	walkErr := filepath.WalkDir(realRoot, func(fullPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !isWithinRealRoot(realRoot, fullPath) {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relOS, err := filepath.Rel(realRoot, fullPath)
		if err != nil {
			return nil
		}
		rel := filepath.ToSlash(relOS)
		if rel == "." {
			return nil
		}

		if ks.IsKeptSubtree(rel) {
			files, bytes := countEntryOrSubtreeStats(fullPath, d)
			stats.KeptFiles += files
			stats.KeptBytes += bytes
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if ks.IsAncestor(rel) {
			return nil
		}

		// Entry is outside the keep-set.
		if normalizedMode == PruneModeTrue {
			files, bytes := countEntryOrSubtreeStats(fullPath, d)
			stats.RemovedFiles += files
			stats.RemovedBytes += bytes
			if d.IsDir() {
				if removeErr := os.RemoveAll(fullPath); removeErr != nil {
					return fmt.Errorf("removing directory %q: %v", rel, removeErr)
				}
				return filepath.SkipDir
			}
			if removeErr := os.Remove(fullPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return fmt.Errorf("removing file %q: %v", rel, removeErr)
			}
			return nil
		}

		// In report-only mode, descend into directories to inspect individual files.
		if d.IsDir() {
			return nil
		}
		stats.RemovedFiles++
		if info, infoErr := d.Info(); infoErr == nil {
			stats.RemovedBytes += info.Size()
		}
		if isUntracedDataFile(rel, schema) && !untracedSeen[rel] {
			untracedSeen[rel] = true
			stats.UntracedDataFiles = append(stats.UntracedDataFiles, rel)
		}
		return nil
	})
	if walkErr != nil {
		return stats, walkErr
	}

	if normalizedMode == PruneModeReportOnly {
		sort.Strings(stats.UntracedDataFiles)
		if err := removeStandaloneDir(realRoot, schema); err != nil {
			return stats, err
		}
	}

	return stats, nil
}

func removeStandaloneDir(realRoot string, schema *BundleSchema) error {
	if schema == nil || strings.TrimSpace(schema.Metadata.DistDir) == "" {
		return nil
	}
	rawStandaloneRel := path.Join(schema.Metadata.DistDir, "standalone")
	standaloneRel, err := CleanAndValidatePath(realRoot, rawStandaloneRel, false)
	if err != nil {
		return fmt.Errorf("validating standalone directory %q: %v", rawStandaloneRel, err)
	}
	if isSubpathOrEqual(standaloneRel, CloudRunDirName) || isSubpathOrEqual(CloudRunDirName, standaloneRel) {
		return fmt.Errorf("refusing to remove standalone directory %q overlapping %s", standaloneRel, CloudRunDirName)
	}
	covered := false
	for _, inc := range schema.OutputFiles.ServerApp.Include {
		cleanInc := path.Clean(inc)
		if cleanInc == "." || cleanInc == CloudRunDirName {
			continue
		}
		if isSubpathOrEqual(standaloneRel, cleanInc) {
			covered = true
			break
		}
	}
	if !covered {
		return nil
	}
	standaloneAbs := filepath.Join(realRoot, filepath.FromSlash(standaloneRel))
	if !isWithinRealRoot(realRoot, standaloneAbs) || standaloneAbs == realRoot {
		return fmt.Errorf("standalone directory %q escapes workspace root", standaloneRel)
	}
	if err := os.RemoveAll(standaloneAbs); err != nil {
		return fmt.Errorf("removing standalone directory %q: %v", standaloneRel, err)
	}
	return nil
}

func countEntryOrSubtreeStats(fullPath string, d fs.DirEntry) (int, int64) {
	if d.IsDir() {
		return countSubtreeStats(fullPath)
	}
	info, err := d.Info()
	if err != nil {
		return 1, 0
	}
	return 1, info.Size()
}

func countSubtreeStats(root string) (int, int64) {
	var files int
	var bytes int64
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		files++
		if info, infoErr := d.Info(); infoErr == nil {
			bytes += info.Size()
		}
		return nil
	})
	return files, bytes
}

var (
	ignoredUntracedDirs = map[string]bool{
		"node_modules": true,
		".next":        true,
		".git":         true,
		".hg":          true,
		".turbo":       true,
		".nx":          true,
		".cloudrun":    true,
		".vscode":      true,
		".idea":        true,
		".github":      true,
		"test":         true,
		"tests":        true,
		"__tests__":    true,
		"__mocks__":    true,
		"e2e":          true,
		"cypress":      true,
		"playwright":   true,
	}

	ignoredConfigBasenames = map[string]bool{
		"package.json":        true,
		"package-lock.json":   true,
		"npm-shrinkwrap.json": true,
		"pnpm-lock.yaml":      true,
		"pnpm-workspace.yaml": true,
		"yarn.lock":           true,
		".yarnrc.yml":         true,
		"bun.lock":            true,
		"bun.lockb":           true,
		"tsconfig.json":       true,
		"tsconfig.base.json":  true,
		"tsconfig.build.json": true,
		"jsconfig.json":       true,
		"turbo.json":          true,
		"nx.json":             true,
		"project.json":        true,
		"apphosting.yaml":     true,
		"components.json":     true,
		"biome.json":          true,
		"biome.jsonc":         true,
		"vercel.json":         true,
		"renovate.json":       true,
		".eslintrc.json":      true,
		".eslintrc.yaml":      true,
		".eslintrc.yml":       true,
		".prettierrc":         true,
		".prettierrc.json":    true,
		".prettierrc.yaml":    true,
		".prettierrc.yml":     true,
		"readme.md":           true,
		"changelog.md":        true,
		"contributing.md":     true,
		"license":             true,
		"license.md":          true,
		"license.txt":         true,
		"next-env.d.ts":       true,
		"docker-compose.yml":  true,
		"docker-compose.yaml": true,
	}

	dataFileExtensions = map[string]bool{
		".json":    true,
		".yaml":    true,
		".yml":     true,
		".toml":    true,
		".csv":     true,
		".tsv":     true,
		".xml":     true,
		".sql":     true,
		".db":      true,
		".sqlite":  true,
		".sqlite3": true,
		".pem":     true,
		".crt":     true,
		".key":     true,
		".wasm":    true,
		".bin":     true,
		".dat":     true,
		".proto":   true,
		".graphql": true,
		".gql":     true,
		".md":      true,
		".mdx":     true,
		".txt":     true,
		".ttf":     true,
		".otf":     true,
		".woff":    true,
		".woff2":   true,
	}
)

func isUntracedDataFile(rel string, schema *BundleSchema) bool {
	if schema == nil {
		return false
	}
	projectDir := schema.Metadata.ProjectDir
	distDir := schema.Metadata.DistDir
	if !isSubpathOrEqual(rel, projectDir) {
		return false
	}
	if distDir != "" && isSubpathOrEqual(rel, distDir) {
		return false
	}
	for _, spec := range schema.OutputFiles.ServerApp.Copy {
		if spec.From != "" && isSubpathOrEqual(rel, spec.From) {
			return false
		}
	}

	segments := strings.Split(rel, "/")
	for _, seg := range segments[:len(segments)-1] {
		if ignoredUntracedDirs[seg] {
			return false
		}
	}

	base := segments[len(segments)-1]
	lowerBase := strings.ToLower(base)
	if ignoredConfigBasenames[lowerBase] {
		return false
	}
	if strings.HasPrefix(lowerBase, "tsconfig.") && strings.HasSuffix(lowerBase, ".json") {
		return false
	}

	ext := strings.ToLower(path.Ext(base))
	return dataFileExtensions[ext]
}
