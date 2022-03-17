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

// Package mockprocessutil provides utils for syncing the mockprocess
// library with the mockprocess binary.
package mockprocessutil

import "encoding/json"

const (
	// EnvHelperMockProcessMap is the env var used to communicate the intended
	// behavior of the mock process for various commands. It contains a
	// map[string]MockProcess serialized to JSON.
	EnvHelperMockProcessMap = "HELPER_MOCK_PROCESS_MAP"
)

// MockProcessConfig encapsulates the behavior of a mock process for test.
// To add more behaviors to the mock process, expand this struct and implement
// the corresponding handling in
// internal/buildpacktest/mockprocess/mockprocess.go
type MockProcessConfig struct {
	// Stdout is the message that should be printed to stdout.
	Stdout string
	// Stderr is the message that should be printed to stderr.
	Stderr string
	// ExitCode is the exit code that the process should use.
	ExitCode int
}

// UnmarshalMockProcessMap is a utility function that marshals a
// map[string]MockProcess from JSON.
func UnmarshalMockProcessMap(data string) (map[string]*MockProcessConfig, error) {
	var mocks map[string]*MockProcessConfig
	if err := json.Unmarshal([]byte(data), &mocks); err != nil {
		return mocks, err
	}

	return mocks, nil
}
