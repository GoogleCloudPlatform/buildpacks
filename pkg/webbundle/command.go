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
	"path"
	"strings"
	"unicode"
)

// ParseCommand splits a run command string into an argument slice suitable for
// ctx.AddWebProcess.
//
// TODO(b/578287546): Move this and the matching parseCommand copies in
// cmd/firebase/bundle/lib and cmd/nodejs/firebasebundle/lib into one shared
// helper. Those copies return gcp.UserErrorf for an unclosed quote and accept an
// empty command, so settle on one behavior when merging them.
func ParseCommand(command string) ([]string, error) {
	var result []string
	var curr strings.Builder
	inQuotes := false

	for _, r := range command {
		if r == '"' {
			inQuotes = !inQuotes
			if !inQuotes {
				result = append(result, curr.String())
				curr.Reset()
			}
			continue
		}
		if unicode.IsSpace(r) && !inQuotes {
			if curr.Len() > 0 {
				result = append(result, curr.String())
				curr.Reset()
			}
			continue
		}
		curr.WriteRune(r)
	}
	if curr.Len() > 0 {
		result = append(result, curr.String())
	}
	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in command %q", command)
	}
	if len(result) == 0 {
		return nil, errors.New("command is empty")
	}
	return result, nil
}

// UsesServeJS reports whether cmd invokes .cloudrun/serve.js.
func UsesServeJS(cmd string) bool {
	for _, field := range strings.Fields(cmd) {
		cleaned := path.Clean(strings.Trim(field, `"'`))
		if cleaned == ".cloudrun/serve.js" || strings.HasSuffix(cleaned, "/.cloudrun/serve.js") {
			return true
		}
	}
	return false
}
