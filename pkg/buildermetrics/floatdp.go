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

// FloatDP is a float64 distribution point measurement to be recorded in BuilderMetrics.  All
// FloatDPs for a given metric name will be added to a distribution as they are aggregated and
// recorded during RCS's GetBuildAnalysis.  Do not use this type for recording non-distribution
// type metrics.
type FloatDP struct {
	value float64
}

// Add increases the value of a FloatDP by addend.  Not threadsafe.
func (f *FloatDP) Add(addend float64) {
	f.value += addend
}

// Value retrieves the value of a FloatDP.  Not threadsafe.
func (f *FloatDP) Value() float64 {
	return f.value
}

// MarshalJSON serializes a FloatDP into json
func (f *FloatDP) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.Value())
}

// UnmarshalJSON deserializes json into a FloatDP
func (f *FloatDP) UnmarshalJSON(b []byte) error {
	var val float64
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	f.value = val
	return nil
}
