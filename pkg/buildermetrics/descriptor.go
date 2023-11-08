// Copyright 2023 Google LLC
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

package buildermetrics

import (
	"fmt"
)

// MetricID is a unique identifier for a metric.
type MetricID string

// Descriptor returns the registered Descriptor for a given MetricID
func (m MetricID) Descriptor() (Descriptor, error) {
	desc, ok := descriptors[m]
	if !ok {
		return Descriptor{}, fmt.Errorf("Descriptor for MetricID %q not found", m)
	}
	if desc.Name == "" || desc.Description == "" {
		return Descriptor{}, fmt.Errorf("Descriptor %q (for MetricID %q) must have both a Name and a Description", desc, m)
	}
	return desc, nil
}

// Descriptor provides details about a metric
type Descriptor struct {
	ID          MetricID // This is useful to have for encoding, to avoid looping through all descriptors to lookup for each stream
	Name        string
	Description string
	Labels      []Label
}

func (d1 Descriptor) equal(d2 Descriptor) bool {
	return d1.ID == d2.ID && d1.Name == d2.Name && d1.Description == d2.Description && labelListsMatch(d1.Labels, d2.Labels)
}

// newDescriptor creates a new Descriptor.
func newDescriptor(id MetricID, name string, description string, labels ...Label) Descriptor {
	return Descriptor{
		ID:          id,
		Name:        name,
		Description: description,
		Labels:      labels,
	}
}
