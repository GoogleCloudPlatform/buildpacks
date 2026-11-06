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
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CleanAndValidatePath cleans rawPath and ensures it is a relative path within
// workspaceRoot without '..' segments or symlinks escaping workspaceRoot.
func CleanAndValidatePath(workspaceRoot, rawPath string, allowDot bool) (string, error) {
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	if strings.ContainsRune(trimmed, '\\') {
		return "", fmt.Errorf("path %q contains backslash", rawPath)
	}
	if strings.HasPrefix(trimmed, "/") || filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("path %q must be relative to workspace root", rawPath)
	}
	for _, seg := range strings.Split(trimmed, "/") {
		if seg == ".." {
			return "", fmt.Errorf("path %q must not contain '..' segments", rawPath)
		}
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		if !allowDot {
			return "", fmt.Errorf("path %q resolves to workspace root '.'", rawPath)
		}
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path %q escapes workspace root", rawPath)
	}

	if err := verifyNoSymlinkEscape(realRoot, cleaned); err != nil {
		return "", err
	}
	return cleaned, nil
}

func resolveRealWorkspaceRoot(workspaceRoot string) (string, error) {
	if strings.TrimSpace(workspaceRoot) == "" {
		return "", errors.New("workspace root is empty")
	}
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("resolving workspace root %q: %w", workspaceRoot, err)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("evaluating symlinks for workspace root %q: %w", workspaceRoot, err)
	}
	return realRoot, nil
}

func verifyNoSymlinkEscape(realRoot, cleanRel string) error {
	parts := strings.Split(cleanRel, "/")
	cur := realRoot
	for _, part := range parts {
		cur = filepath.Join(cur, part)
		var err error
		cur, err = resolveComponentWithinRoot(realRoot, cur, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

const maxSymlinkDepth = 40

func resolveComponentWithinRoot(realRoot, candidateAbs string, depth int) (string, error) {
	if depth > maxSymlinkDepth {
		return "", fmt.Errorf("too many levels of symbolic links at %q", candidateAbs)
	}
	if !isWithinRealRoot(realRoot, candidateAbs) {
		return "", fmt.Errorf("path %q escapes workspace root %q", candidateAbs, realRoot)
	}

	info, err := os.Lstat(candidateAbs)
	if os.IsNotExist(err) {
		return candidateAbs, nil
	}
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", candidateAbs, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return candidateAbs, nil
	}

	linkTarget, err := os.Readlink(candidateAbs)
	if err != nil {
		return "", fmt.Errorf("readlink %q: %w", candidateAbs, err)
	}
	var resolvedTarget string
	if filepath.IsAbs(linkTarget) {
		resolvedTarget = filepath.Clean(linkTarget)
	} else {
		resolvedTarget = filepath.Clean(filepath.Join(filepath.Dir(candidateAbs), linkTarget))
	}
	if !isWithinRealRoot(realRoot, resolvedTarget) {
		return "", fmt.Errorf("symlink %q -> %q escapes workspace root %q", candidateAbs, linkTarget, realRoot)
	}

	relTarget, err := filepath.Rel(realRoot, resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("computing relative path for %q: %w", resolvedTarget, err)
	}
	if relTarget == "." {
		return realRoot, nil
	}
	targetParts := strings.Split(filepath.ToSlash(relTarget), "/")
	walkCur := realRoot
	for _, part := range targetParts {
		walkCur = filepath.Join(walkCur, part)
		walkCur, err = resolveComponentWithinRoot(realRoot, walkCur, depth+1)
		if err != nil {
			return "", err
		}
	}
	return walkCur, nil
}

func isWithinRealRoot(realRoot, targetAbs string) bool {
	cleanRoot := filepath.Clean(realRoot)
	cleanTarget := filepath.Clean(targetAbs)
	if cleanTarget == cleanRoot {
		return true
	}
	prefix := cleanRoot
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(cleanTarget, prefix)
}

func isSubpathOrEqual(candidate, parent string) bool {
	candidate = path.Clean(candidate)
	parent = path.Clean(parent)
	if parent == "." {
		return candidate != ".." && !strings.HasPrefix(candidate, "../")
	}
	return candidate == parent || strings.HasPrefix(candidate, parent+"/")
}

func isCoveredByInclude(candidate string, include []string) bool {
	for _, inc := range include {
		if isSubpathOrEqual(candidate, inc) {
			return true
		}
	}
	return false
}
