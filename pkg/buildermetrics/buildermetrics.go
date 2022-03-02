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

// Package buildermetrics provides functionality to write metrics to builderoutput.
package buildermetrics

import (
	"encoding/json"
	"sync"
)

var (
	bm   *BuilderMetrics
	mu   sync.Mutex
	once sync.Once
)

// BuilderMetrics contains the metrics to be reported to RCS via BuilderOutput
type BuilderMetrics struct {
	counters map[CounterID]*Counter
}

// NewBuilderMetrics returns a new, empty BuilderMetrics
// For testing use only
func NewBuilderMetrics() BuilderMetrics {
	return BuilderMetrics{make(map[CounterID]*Counter)}
}

// GetCounter returns the Counter with MetricID m, or creates it
func (b *BuilderMetrics) GetCounter(m CounterID) *Counter {
	if _, found := b.counters[m]; !found {
		b.counters[m] = &Counter{}
	}
	return b.counters[m]
}

// Reset resets the state of the metrics struct
// For testing use only.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	bm = &BuilderMetrics{make(map[CounterID]*Counter)}
}

// GlobalBuilderMetrics returns a pointer to the BuilderMetrics singleton
func GlobalBuilderMetrics() *BuilderMetrics {
	mu.Lock()
	defer mu.Unlock()
	once.Do(
		func() {
			bm = &BuilderMetrics{make(map[CounterID]*Counter)}
		})
	return bm
}

type countersMap struct {
	Counters map[CounterID]*Counter `json:"c,omitempty"`
}

// MarshalJSON is a custom marshaler for BuilderMetrics
func (b BuilderMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(countersMap{Counters: b.counters})
}

// UnmarshalJSON is a custom unmarshaler for BuilderMetrics
func (b *BuilderMetrics) UnmarshalJSON(j []byte) error {
	var val countersMap
	if err := json.Unmarshal(j, &val); err != nil {
		return err
	}
	if val.Counters == nil {
		b.counters = make(map[CounterID]*Counter)
	} else {
		b.counters = val.Counters
	}
	return nil
}
