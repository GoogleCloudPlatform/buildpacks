// Copyright 2025 Google LLC
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

// Package buildermetadata provides an interface for builder metadata with text-based metrics.
package buildermetadata

import (
	"encoding/json"
	"sync"
)

var (
	bmd  *BuilderMetadata
	mu   sync.Mutex
	once sync.Once
)

// BuilderMetadata contains the builder metadata to be reported to RCS via BuilderOutput.
type BuilderMetadata struct {
	metadata map[MetadataID]MetadataValue
}

// NewBuilderMetadata returns a new, empty BuilderMetadata
func NewBuilderMetadata() BuilderMetadata {
	return BuilderMetadata{make(map[MetadataID]MetadataValue)}
}

// MetadataValue is the metadata value corresponding to the MetadataID.
type MetadataValue string

// MetadataID is the unique identifier for each metadata value.
type MetadataID string

// The MetadataID enum below define new metadata that can be recorded.
// To add a new value, add a new const MetadataID below and
// the buildermetadata package will be able to track new metadata.
//
// Intended usage:
//
//	buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.IsUsingGenkit, "true")
const (
	IsUsingGenkit    MetadataID = "1" // Whether the application is using Genkit
	IsUsingGenAI     MetadataID = "2" // Whether the application is using GenAI API
	FrameworkName    MetadataID = "3" // The framework name used in the build.
	FrameworkVersion MetadataID = "4" // The framework version used in the build.
	AdapterName      MetadataID = "5" // The adapter name used in the build.
	AdapterVersion   MetadataID = "6" // The adapter version used in the build.
	MonorepoName     MetadataID = "7" // The monorepo name used in the build.
)

// MetadataIDNames is a lookup map from MetadataID to its name.
var MetadataIDNames = map[MetadataID]string{
	IsUsingGenkit:    "IsUsingGenkit",
	IsUsingGenAI:     "IsUsingGenAI",
	FrameworkName:    "FrameworkName",
	FrameworkVersion: "FrameworkVersion",
	AdapterName:      "AdapterName",
	AdapterVersion:   "AdapterVersion",
	MonorepoName:     "MonorepoName",
}

// GetValue returns the Metadata value with MetadataID id, or creates it if it doesn't exist.
func (b *BuilderMetadata) GetValue(id MetadataID) MetadataValue {
	if _, found := b.metadata[id]; !found {
		b.metadata[id] = MetadataValue("false")
	}
	return b.metadata[id]
}

// IsEmpty checks if the BuilderMetadata is empty.
func (b *BuilderMetadata) IsEmpty() bool {
	return b == nil || len(b.metadata) == 0
}

// SetValue sets the Metadata value with MetadataID id.
func (b *BuilderMetadata) SetValue(id MetadataID, value MetadataValue) {
	b.metadata[id] = value
}

// ForEachValue iterates over all values in the BuilderMetadata.
func (b *BuilderMetadata) ForEachValue(f func(MetadataID, MetadataValue)) {
	for id, value := range b.metadata {
		f(id, value)
	}
}

// Reset resets the state of the BuilderMetadata struct
// For testing use only.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	bmd = &BuilderMetadata{make(map[MetadataID]MetadataValue)}
}

// GlobalBuilderMetadata returns a pointer to the BuilderMetadata singleton
func GlobalBuilderMetadata() *BuilderMetadata {
	mu.Lock()
	defer mu.Unlock()
	once.Do(
		func() {
			bmd = &BuilderMetadata{make(map[MetadataID]MetadataValue)}
		})
	return bmd
}

type metadataMaps struct {
	Metadata map[MetadataID]MetadataValue `json:"m,omitempty"`
}

// MarshalJSON is a custom marshaler for BuilderMetadata
func (b BuilderMetadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(metadataMaps{Metadata: b.metadata})
}

// UnmarshalJSON is a custom unmarshaller for BuilderMetadata
func (b *BuilderMetadata) UnmarshalJSON(j []byte) error {
	var val metadataMaps
	if err := json.Unmarshal(j, &val); err != nil {
		return err
	}
	if val.Metadata == nil {
		b.metadata = make(map[MetadataID]MetadataValue)
	} else {
		b.metadata = val.Metadata
	}
	return nil
}
