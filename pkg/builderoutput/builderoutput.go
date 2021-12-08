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

// Package builderoutput provides an interface for serializing BuilderOutput.
package builderoutput

import (
	"encoding/json"
	"fmt"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// FromJSON parses json bytes to a BuilderOutput.
func FromJSON(bytes []byte) (BuilderOutput, error) {
	var bout BuilderOutput
	if err := json.Unmarshal(bytes, &bout); err != nil {
		return BuilderOutput{}, fmt.Errorf("unmarshalling json: %w", err)
	}
	return bout, nil
}

// JSON encodes a BuilderOutput as json
func (bo BuilderOutput) JSON() ([]byte, error) {
	bytes, err := json.Marshal(bo)
	if err != nil {
		return nil, fmt.Errorf("marshalling json: %w", err)
	}
	return bytes, nil
}

// BuilderOutput contains data about the outcome of a build
type BuilderOutput struct {
	Error       buildererror.Error `json:"error"`
	Stats       []BuilderStat      `json:"stats"`
	Warnings    []string           `json:"warnings"`
	CustomImage bool               `json:"customImage"`
}

// IsSystemError determines if the error type is a SYSTEM-attributed error
func (bo BuilderOutput) IsSystemError() bool {
	return bo.Error.Type == buildererror.StatusInternal
}

// BuilderStat contains statistics about a build step
type BuilderStat struct {
	BuildpackID      string `json:"buildpackId"`
	BuildpackVersion string `json:"buildpackVersion"`
	DurationMs       int64  `json:"totalDurationMs"`
	UserDurationMs   int64  `json:"userDurationMs"`
}
