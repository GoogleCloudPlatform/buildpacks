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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// MergeCopy executes the copy specifications in order, merging directories and
// overwriting destination files. Missing source paths are skipped silently.
func MergeCopy(workspaceRoot string, specs []CopySpec) error {
	realRoot, err := resolveRealWorkspaceRoot(workspaceRoot)
	if err != nil {
		return err
	}
	for _, spec := range specs {
		fromRel, err := CleanAndValidatePath(realRoot, spec.From, false)
		if err != nil {
			return fmt.Errorf("invalid copy.from %q: %v", spec.From, err)
		}
		toRel, err := CleanAndValidatePath(realRoot, spec.To, false)
		if err != nil {
			return fmt.Errorf("invalid copy.to %q: %v", spec.To, err)
		}
		if isSubpathOrEqual(toRel, fromRel) || isSubpathOrEqual(fromRel, toRel) {
			return fmt.Errorf("copy.from %q and copy.to %q must not overlap", fromRel, toRel)
		}

		srcAbs := filepath.Join(realRoot, filepath.FromSlash(fromRel))
		dstAbs := filepath.Join(realRoot, filepath.FromSlash(toRel))

		srcInfo, err := os.Lstat(srcAbs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("stat copy source %q: %v", fromRel, err)
		}

		if err := mergeNode(realRoot, srcAbs, dstAbs, srcInfo); err != nil {
			return fmt.Errorf("copying %q to %q: %v", fromRel, toRel, err)
		}
	}
	return nil
}

func mergeNode(realRoot, srcAbs, dstAbs string, srcInfo fs.FileInfo) error {
	if !isWithinRealRoot(realRoot, srcAbs) || !isWithinRealRoot(realRoot, dstAbs) {
		return fmt.Errorf("copy path escapes workspace root")
	}

	mode := srcInfo.Mode()
	switch {
	case mode&os.ModeSymlink != 0:
		if _, err := resolveComponentWithinRoot(realRoot, srcAbs, 0); err != nil {
			return err
		}
		linkTarget, err := os.Readlink(srcAbs)
		if err != nil {
			return err
		}
		dstLinkTarget := linkTarget
		if !filepath.IsAbs(linkTarget) {
			targetAbs := filepath.Clean(filepath.Join(filepath.Dir(srcAbs), linkTarget))
			relTarget, relErr := filepath.Rel(filepath.Dir(dstAbs), targetAbs)
			if relErr != nil {
				return relErr
			}
			dstLinkTarget = relTarget
		}
		if err := os.MkdirAll(filepath.Dir(dstAbs), 0755); err != nil {
			return err
		}
		if _, err := os.Lstat(dstAbs); err == nil {
			if err := os.RemoveAll(dstAbs); err != nil {
				return err
			}
		}
		if err := os.Symlink(dstLinkTarget, dstAbs); err != nil {
			return err
		}
		if _, err := resolveComponentWithinRoot(realRoot, dstAbs, 0); err != nil {
			_ = os.Remove(dstAbs)
			return err
		}
		return nil

	case srcInfo.IsDir():
		if dstInfo, err := os.Lstat(dstAbs); err == nil && !dstInfo.IsDir() {
			if err := os.Remove(dstAbs); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(dstAbs, 0755); err != nil {
			return err
		}
		entries, err := os.ReadDir(srcAbs)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			childSrc := filepath.Join(srcAbs, entry.Name())
			childDst := filepath.Join(dstAbs, entry.Name())
			childInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if err := mergeNode(realRoot, childSrc, childDst, childInfo); err != nil {
				return err
			}
		}
		return nil

	case mode.IsRegular():
		if err := os.MkdirAll(filepath.Dir(dstAbs), 0755); err != nil {
			return err
		}
		if dstInfo, err := os.Lstat(dstAbs); err == nil && (dstInfo.IsDir() || dstInfo.Mode()&os.ModeSymlink != 0) {
			if err := os.RemoveAll(dstAbs); err != nil {
				return err
			}
		}
		return copyRegularFile(srcAbs, dstAbs, mode.Perm())

	default:
		return nil
	}
}

func copyRegularFile(srcAbs, dstAbs string, perm fs.FileMode) error {
	in, err := os.Open(srcAbs)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dstAbs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
