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
// THIS PACKAGE IS NOT THREADSAFE.
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
	floatDPs map[FloatDPID]*FloatDP
}

// NewBuilderMetrics returns a new, empty BuilderMetrics
// For testing use only
func NewBuilderMetrics() BuilderMetrics {
	return BuilderMetrics{make(map[CounterID]*Counter), make(map[FloatDPID]*FloatDP)}
}

// GetCounter returns the Counter with CounterID m, or creates it
func (b *BuilderMetrics) GetCounter(m CounterID) *Counter {
	if _, found := b.counters[m]; !found {
		b.counters[m] = &Counter{}
	}
	return b.counters[m]
}

// ForEachCounter executes a function for each initialized Counter
func (b *BuilderMetrics) ForEachCounter(f func(CounterID, *Counter)) {
	for id, c := range b.counters {
		f(id, c)
	}
}

// GetFloatDP returns the FloatDP with FloatDPID m, or creates it
func (b *BuilderMetrics) GetFloatDP(m FloatDPID) *FloatDP {
	if _, found := b.floatDPs[m]; !found {
		b.floatDPs[m] = &FloatDP{}
	}
	return b.floatDPs[m]
}

// ForEachFloatDP executes a function for each initialized FloatDP
func (b *BuilderMetrics) ForEachFloatDP(f func(FloatDPID, *FloatDP)) {
	for id, fm := range b.floatDPs {
		f(id, fm)
	}
}

// Reset resets the state of the metrics struct
// For testing use only.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	bm = &BuilderMetrics{make(map[CounterID]*Counter), make(map[FloatDPID]*FloatDP)}
}

// GlobalBuilderMetrics returns a pointer to the BuilderMetrics singleton
func GlobalBuilderMetrics() *BuilderMetrics {
	mu.Lock()
	defer mu.Unlock()
	once.Do(
		func() {
			bm = &BuilderMetrics{make(map[CounterID]*Counter), make(map[FloatDPID]*FloatDP)}
		})
	return bm
}

type metricsMaps struct {
	Counters map[CounterID]*Counter `json:"c,omitempty"`
	FloatDPs map[FloatDPID]*FloatDP `json:"f,omitempty"`
}

// MarshalJSON is a custom marshaler for BuilderMetrics
func (b BuilderMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(metricsMaps{Counters: b.counters, FloatDPs: b.floatDPs})
}

// UnmarshalJSON is a custom unmarshaller for BuilderMetrics
func (b *BuilderMetrics) UnmarshalJSON(j []byte) error {
	var val metricsMaps
	if err := json.Unmarshal(j, &val); err != nil {
		return err
	}
	if val.Counters == nil {
		b.counters = make(map[CounterID]*Counter)
	} else {
		b.counters = val.Counters
	}
	if val.FloatDPs == nil {
		b.floatDPs = make(map[FloatDPID]*FloatDP)
	} else {
		b.floatDPs = val.FloatDPs
	}
	return nil
}
