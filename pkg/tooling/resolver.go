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

// Package tooling provides configuration related to pre-installed build tools.
package tooling

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// parsedToolingData stores the unmarshaled content of toolingVersions.
var parsedToolingData map[string]LanguageInfo
var parseError error
var once sync.Once

//go:embed tooling_versions.json
var toolingVersions []byte

func parseToolingVersions() {
	parseError = json.Unmarshal(toolingVersions, &parsedToolingData)
}

// RuntimeInfo stores version information for specific tools related to a runtime.
type RuntimeInfo struct {
	Names  []string          `json:"names,omitempty"`
	Stacks []string          `json:"stacks,omitempty"`
	Tools  map[string]string `json:"tools,omitempty"`
}

// LanguageInfo stores default and runtime-specific tool version mappings for a language.
type LanguageInfo struct {
	Default  map[string]string `json:"default,omitempty"`
	Runtimes []RuntimeInfo     `json:"runtimes,omitempty"`
}

// ResolveToolVersion resolves a pinned version based on language, runtime, stack, and toolName from tooling_versions.json.
func ResolveToolVersion(language, toolName, runtimeVersion, stackID string) (string, error) {
	once.Do(parseToolingVersions)
	if parseError != nil {
		return "", fmt.Errorf("parsing tooling_versions.json: %w", parseError)
	}

	langInfo, ok := parsedToolingData[language]
	if !ok {
		return "", fmt.Errorf("language %q not found in TOOLING_VERSIONS", language)
	}

	runtimeName := ""
	parts := strings.Split(runtimeVersion, ".")
	major := parts[0]
	minor := ""
	if len(parts) >= 2 {
		minor = parts[1]
	}
	switch language {
	case "java", "nodejs", "dotnet":
		runtimeName = language + major
	case "python", "go", "ruby", "php":
		runtimeName = language + major + minor
	default:
		runtimeName = language + major + minor
	}

	// 1. Check for specific runtime overrides
	for _, rtInfo := range langInfo.Runtimes {
		matchName := false
		for _, name := range rtInfo.Names {
			if name == runtimeName {
				matchName = true
				break
			}
		}

		matchStack := false
		for _, stack := range rtInfo.Stacks {
			if stack == stackID {
				matchStack = true
				break
			}
		}

		if matchName || matchStack {
			if ver, ok := rtInfo.Tools[toolName]; ok {
				return ver, nil
			}
		}
	}

	// 2. Fall back to the default tools
	if ver, ok := langInfo.Default[toolName]; ok {
		return ver, nil
	}

	return "", fmt.Errorf("tool %q not found for language %q with runtime %q and stack %q", toolName, language, runtimeVersion, stackID)
}
