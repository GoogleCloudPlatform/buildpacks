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
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIncrementCounter(t *testing.T) {
	Reset()

	GlobalBuilderMetrics().GetCounter(ArNpmCredsGenCounterID).Increment(3)
	GlobalBuilderMetrics().GetCounter(ArNpmCredsGenCounterID).Increment(6)
	if GlobalBuilderMetrics().GetCounter(ArNpmCredsGenCounterID).Value() != 9 {
		t.Errorf("ArNpmCredsGenCount not successfully stored")
	}
}

func TestBuilderMetricsUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  BuilderMetrics
	}{
		{
			name:  "basic",
			input: []byte(`{"c":{"1":3}}`),
			want:  BuilderMetrics{map[CounterID]*Counter{"1": &Counter{3}}},
		},
		{
			name:  "empty",
			input: []byte(`{}`),
			want:  BuilderMetrics{map[CounterID]*Counter{}},
		},
		{
			name:  "multiple",
			input: []byte(`{"c":{"1":3,"2":18}}`),
			want:  BuilderMetrics{map[CounterID]*Counter{"1": &Counter{3}, "2": &Counter{18}}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got BuilderMetrics
			err := json.Unmarshal(tc.input, &got)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(BuilderMetrics{}, Counter{})); diff != "" {
				t.Errorf("BuilderMetrics json.Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuilderMetricsMarshalJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input BuilderMetrics
		want  []byte
	}{
		{
			name:  "basic",
			input: BuilderMetrics{map[CounterID]*Counter{"1": &Counter{3}}},
			want:  []byte(`{"c":{"1":3}}`),
		},
		{
			name:  "empty",
			input: BuilderMetrics{map[CounterID]*Counter{}},
			want:  []byte(`{}`),
		},
		{
			name:  "multiple",
			input: BuilderMetrics{map[CounterID]*Counter{"1": &Counter{3}, "2": &Counter{18}}},
			want:  []byte(`{"c":{"1":3,"2":18}}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j, err := json.Marshal(&tc.input)

			if err != nil {
				t.Fatalf("MarshalJSON test name: %v, %v: %v", tc.name, tc.input, err)
			}
			if !bytes.Equal(tc.want, j) {
				t.Errorf("test name: %v got %v, want %v", tc.name, string(j), string(tc.want))
			}
		})
	}
}

func TestForEachCounter(t *testing.T) {
	b := NewBuilderMetrics()
	c1 := b.GetCounter("1")
	c2 := b.GetCounter("2")
	c3 := b.GetCounter("3")
	c1.Increment(3)
	c2.Increment(5)
	c3.Increment(8)
	sum := int64(0)
	b.ForEachCounter(func(id CounterID, c *Counter) {
		sum += c.Value()
	})
	want := c1.Value() + c2.Value() + c3.Value()
	if sum != want {
		t.Errorf("ForEachCounter counter.Value() sum = %v, want %v", sum, want)
	}
}
