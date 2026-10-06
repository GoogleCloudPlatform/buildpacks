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
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
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
	Fetch                    *FetchOutput                    `json:"fetch,omitempty"`
	InstalledRuntimeVersions []string                        `json:"rtVersions,omitempty"`
	Metrics                  buildermetrics.BuilderMetrics   `json:"metrics"`
	Error                    buildererror.Error              `json:"error"`
	Metadata                 buildermetadata.BuilderMetadata `json:"metadata"`
	Stats                    []BuilderStat                   `json:"stats"`
	Warnings                 []string                        `json:"warnings"`
	CustomImage              bool                            `json:"customImage"`
}

// FetchOutput captures detailed metrics and status of the source fetch step.
type FetchOutput struct {
	Status             string              `json:"status"`
	SourceType         string              `json:"sourceType"`
	Location           string              `json:"location"`
	TotalDurationMs    int64               `json:"totalDurationMs"`
	DownloadDurationMs int64               `json:"downloadDurationMs"`
	UnzipDurationMs    int64               `json:"unzipDurationMs"`
	DownloadBytes      int64               `json:"downloadBytes"`
	FilesCount         int                 `json:"filesCount"`
	Error              *buildererror.Error `json:"error,omitempty"`
}

// New constructs a BuilderOutput and returns a pointer.
func New() *BuilderOutput {
	return &BuilderOutput{
		Metrics:  buildermetrics.NewBuilderMetrics(),
		Metadata: buildermetadata.NewBuilderMetadata(),
	}
}

// IsSystemError determines if the error type is a SYSTEM-attributed error
func (bo BuilderOutput) IsSystemError() bool {
	if bo.Fetch != nil && bo.Fetch.Error != nil && bo.Fetch.Error.Type == buildererror.StatusInternal {
		return true
	}
	return bo.Error.Type == buildererror.StatusInternal
}

// BuilderStat contains statistics about a build step
type BuilderStat struct {
	BuildpackID      string `json:"buildpackId"`
	BuildpackVersion string `json:"buildpackVersion"`
	DurationMs       int64  `json:"totalDurationMs"`
	UserDurationMs   int64  `json:"userDurationMs"`
}
