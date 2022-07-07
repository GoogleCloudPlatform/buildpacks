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

// Package testdata contains utilities for working with test data.
package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// MustGetPath returns the absolute path of a testdata file.
// The reason to have this function is blaze runs all tests from the runfiles directory and we set
// our corresponding bazel go_test.rundir to the runfiles directory as well. This breaks the standard
// go convention of running the test from the test's source directory. The goal of this function is
// to enable tests to access files in ${TEST_WORKING_DIRECTORY}/testdata/ as normal, with only needing
// a one liner to this function.
//
// Since tests may want to call this function when initializing a package level 'var', this function
// panics on error instead of returning an error or requiring the test to passing a *testing.T.
func MustGetPath(relativePath string) string {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("getting current working directory: %v", err))
	}

	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("unable to get caller information")
	}
	if !strings.HasSuffix(filename, "_test.go") {
		panic(fmt.Sprintf("invalid caller source file name '%v': MustGetPath() must be invoked from a test", filepath.Base(filename)))
	}
	return filepath.Join(wd, filepath.Dir(filename), relativePath)
}
