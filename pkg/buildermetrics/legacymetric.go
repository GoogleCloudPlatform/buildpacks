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

package buildermetrics

import (
	"encoding/json"
)

// LegacyCounter is an increasing metric to be recorded in BuilderMetrics.
type LegacyCounter legacyMetric[int64]

// LegacyFloatDP is a float64 distribution point measurement to be recorded in BuilderMetrics.  All
// LegacyFloatDPs for a given metric name should be added to a single distribution as they are
// aggregated and recorded when processing BuilderOutput.  Do not use this type for recording
// non-distribution type metrics.
//
// Also, keep in mind that these are floats, and care must be taken with comparisons and overflow.
type LegacyFloatDP legacyMetric[float64]

type legacyMetric[T interface{ int64 | float64 }] struct {
	value T
}

// Add increases the value of a Counter by addend
func (l *legacyMetric[T]) Add(addend T) {
	l.value += addend
}

// Value retrieves the value of a Counter
func (l *legacyMetric[T]) Value() T {
	return l.value
}

// Legacy metrics must un/marshal in the same way as the original metrics for compatibility

// MarshalJSON serializes a Counter into json
func (l *legacyMetric[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.value)
}

// UnmarshalJSON deserializes json into a Counter
func (l *legacyMetric[T]) UnmarshalJSON(b []byte) error {
	var val T
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	l.value = val
	return nil
}
