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

package gcpbuildpack

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

type spanInfo struct {
	name       string
	start      time.Time
	end        time.Time
	attributes map[string]interface{}
	status     buildererror.Status
}

func newSpanInfo(name string, start, end time.Time, attributes map[string]interface{}, status buildererror.Status) (*spanInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("span name required")
	}
	if start.After(end) {
		return nil, fmt.Errorf("start is after end")
	}

	// TODO: validate attributes
	// See https://cloud.google.com/trace/docs/reference/v2/rest/v2/Attributes

	return &spanInfo{
		name:       name,
		start:      start,
		end:        end,
		attributes: attributes,
		status:     status,
	}, nil
}

func (ctx *Context) createSpanName(cmd []string) string {
	var trimmed []string
	for _, c := range cmd {
		t := strings.TrimSpace(c)
		if t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return fmt.Sprintf("Exec %q", strings.Join(trimmed, " "))
}
