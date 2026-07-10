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

// Package entrypoint provides functions to parse Procfiles.
package entrypoint

import (
	"regexp"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	processRe = regexp.MustCompile(`(?m)^(\w+):\s*(.+)$`)
)

// Parse parses a Procfile content and returns a map of process names to commands.
// If multiple processes have the same name, only the first one is returned.
func Parse(ctx *gcp.Context, content string) (map[string]string, error) {
	matches := processRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, gcp.UserErrorf("did not find any processes in Procfile")
	}

	processes := make(map[string]string)
	for _, match := range matches {
		// Sanity check, if this fails there is a mistake in the regex.
		// One group for overall match and two subgroups.
		if len(match) != 3 {
			return nil, gcp.InternalErrorf("invalid process match, want slice of two strings, got: %v", match)
		}
		name, command := match[1], strings.TrimSpace(match[2])
		if _, exists := processes[name]; exists {
			ctx.Warnf("Skipping duplicate %s process: %s", name, command)
			continue
		}
		processes[name] = command
	}
	return processes, nil
}
