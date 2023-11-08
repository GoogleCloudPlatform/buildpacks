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

func TestAddFloatDP(t *testing.T) {
	Reset()

	GlobalBuilderMetrics().GetFloatDP(ComposerInstallLatencyID).Add(18.3)
	GlobalBuilderMetrics().GetFloatDP(ComposerInstallLatencyID).Add(3)
	if GlobalBuilderMetrics().GetFloatDP(ComposerInstallLatencyID).Value() != 21.3 {
		t.Errorf("ComposerInstallLatencyID not successfully stored")
	}
}

func TestBuilderMetricsUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  BuilderMetrics
	}{
		{
			name:  "empty",
			input: []byte(`{}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{},
				map[MetricID]*FloatDP{},
			},
		},
		{
			name:  "basic counter",
			input: []byte(`{"c":{"1":3}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{"1": &Counter{3}},
				map[MetricID]*FloatDP{},
			},
		},
		{
			name:  "multiple counter",
			input: []byte(`{"c":{"1":3,"2":18}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{"1": &Counter{3}, "2": &Counter{18}},
				map[MetricID]*FloatDP{},
			},
		},
		{
			name:  "basic float",
			input: []byte(`{"f":{"1":3}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{},
				map[MetricID]*FloatDP{"1": &FloatDP{3}},
			},
		},
		{
			name:  "multiple float",
			input: []byte(`{"f":{"1":3.3,"2":18.18}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{},
				map[MetricID]*FloatDP{"1": &FloatDP{3.3}, "2": &FloatDP{18.18}},
			},
		},
		{
			name:  "both",
			input: []byte(`{"c":{"1":3}, "f":{"1":3.3,"2":18.18}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{"1": &Counter{3}},
				map[MetricID]*FloatDP{"1": &FloatDP{3.3}, "2": &FloatDP{18.18}},
			},
		},
		{
			name:  "unordered",
			input: []byte(`{"f":{"1":3.3,"2":18.18}, "c":{"1":3}}`),
			want: BuilderMetrics{
				map[MetricID]*Counter{"1": &Counter{3}},
				map[MetricID]*FloatDP{"1": &FloatDP{3.3}, "2": &FloatDP{18.18}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got BuilderMetrics
			err := json.Unmarshal(tc.input, &got)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(BuilderMetrics{}, Counter{}, FloatDP{})); diff != "" {
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
			name:  "empty",
			input: BuilderMetrics{map[MetricID]*Counter{}, map[MetricID]*FloatDP{}},
			want:  []byte(`{}`),
		},
		{
			name:  "basic counter",
			input: BuilderMetrics{map[MetricID]*Counter{"1": &Counter{3}}, map[MetricID]*FloatDP{}},
			want:  []byte(`{"c":{"1":3}}`),
		},
		{
			name:  "multiple counter",
			input: BuilderMetrics{map[MetricID]*Counter{"1": &Counter{3}, "2": &Counter{18}}, map[MetricID]*FloatDP{}},
			want:  []byte(`{"c":{"1":3,"2":18}}`),
		},
		{
			name:  "basic float",
			input: BuilderMetrics{map[MetricID]*Counter{}, map[MetricID]*FloatDP{"1": &FloatDP{3.3}}},
			want:  []byte(`{"f":{"1":3.3}}`),
		},
		{
			name:  "multiple float",
			input: BuilderMetrics{map[MetricID]*Counter{}, map[MetricID]*FloatDP{"1": &FloatDP{3.3}, "2": &FloatDP{18.18}}},
			want:  []byte(`{"f":{"1":3.3,"2":18.18}}`),
		},
		{
			name:  "both",
			input: BuilderMetrics{map[MetricID]*Counter{"1": &Counter{3}}, map[MetricID]*FloatDP{"1": &FloatDP{3.3}, "2": &FloatDP{18.18}}},
			want:  []byte(`{"c":{"1":3},"f":{"1":3.3,"2":18.18}}`),
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
	b.ForEachCounter(func(id MetricID, c *Counter) {
		sum += c.Value()
	})
	want := c1.Value() + c2.Value() + c3.Value()
	if sum != want {
		t.Errorf("ForEachCounter counter.Value() sum = %v, want %v", sum, want)
	}
}

func TestForEachFloatDPMetric(t *testing.T) {
	b := NewBuilderMetrics()
	f1 := b.GetFloatDP("1")
	f2 := b.GetFloatDP("2")
	f3 := b.GetFloatDP("3")
	f1.Add(3.3)
	f2.Add(5.5)
	f3.Add(8.8)
	sum := float64(0)
	b.ForEachFloatDP(func(id MetricID, f *FloatDP) {
		sum += f.Value()
	})
	want := f1.Value() + f2.Value() + f3.Value()
	if sum != want {
		t.Errorf("ForEachFloatDP floatDP.Value() sum = %v, want %v", sum, want)
	}
}
