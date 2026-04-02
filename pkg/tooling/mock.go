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

package tooling

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed tooling_versions_mock.json
var toolingVersionsMock []byte

// MockData mocks the tooling versions json file for testing.
func MockData() func() {
	once.Do(parseToolingVersions)
	mockJSON := toolingVersionsMock

	originalParsedData := parsedToolingData
	originalParseError := parseError

	parsedToolingData = nil
	parseError = json.Unmarshal(mockJSON, &parsedToolingData)
	if parseError != nil {
		panic(fmt.Sprintf("Failed to unmarshal mock json: %v", parseError))
	}

	return func() {
		parsedToolingData = originalParsedData
		parseError = originalParseError
	}
}
