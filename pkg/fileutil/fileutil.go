// Copyright 2022 Google LLC
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

// Package fileutil contains utilities for filesystem operations.
package fileutil

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type action string

const (
	move action = "move"
	copy action = "copy"
)

// AllPaths indicates all paths should be recursively walked for functions
// that walk the filesystem.
var AllPaths = func(path string, d fs.DirEntry) (bool, error) {
	return true, nil
}

// MaybeCopyPathContents recursively copies the contents of srcPath to destPath.
func MaybeCopyPathContents(destPath, srcPath string, copyCondition func(path string, d fs.DirEntry) (bool, error)) error {
	return moveOrCopyPath(copy, destPath, srcPath, copyCondition)
}

// MaybeMovePathContents moves the contents of srcPath to destPath.
func MaybeMovePathContents(destPath, srcPath string, moveCondition func(path string, d fs.DirEntry) (bool, error)) error {
	return moveOrCopyPath(move, destPath, srcPath, moveCondition)
}

// moveOrCopyPath recursively copies or moves files and directories: from srcPath to destPath.
func moveOrCopyPath(moveOrCopy action, destPath, srcPath string, condition func(path string, d fs.DirEntry) (bool, error)) error {
	return filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root
		if path == srcPath {
			return nil
		}

		shouldCopy, err := condition(path, d)
		if err != nil {
			return err
		}

		if !shouldCopy {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		dest := filepath.Join(destPath, relPath)

		if moveOrCopy == move {
			if err := os.Rename(path, dest); err != nil {
				return err
			}
			// Rename moves the entire directory, so don't need to continue
			// walking the directory.
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return os.MkdirAll(dest, 0744)
		}

		return CopyFile(dest, path)
	})
}

// CopyFile copies a file from src to dest
func CopyFile(dest, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
