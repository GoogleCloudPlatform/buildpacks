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
	"sync/atomic"
)

// CounterID is an index to Counter
type CounterID string

// Counter is an increasing metric to be recorded in BuilderMetrics
type Counter struct {
	value int64
}

// Increment increases the value of a Counter by addend
func (c *Counter) Increment(addend int64) {
	atomic.AddInt64(&c.value, addend)
}

// Value retreives the value of a Counter
func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// MarshalJSON serializes a Counter into json
func (c *Counter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}

// UnmarshalJSON unserializes json into a Counter
func (c *Counter) UnmarshalJSON(b []byte) error {
	var val int64
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	c.value = val
	return nil
}
