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

func TestLegacyCounterUnmarshalJSON(t *testing.T) {
	var got legacyMetric[int64]
	err := json.Unmarshal([]byte(`3`), &got)
	if err != nil {
		t.Fatalf("legacyMetric[int64].UnmarshalJSON: %v", err)
	}

	want := legacyMetric[int64]{3}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(legacyMetric[int64]{}, legacyMetric[float64]{})); diff != "" {
		t.Errorf("legacyMetric[int64].UnmarshalJSON:  diff (-want +got):\n %v", diff)
	}
}

func TestLegacyCounterMarshalJSON(t *testing.T) {
	c := legacyMetric[int64]{3}

	got, err := json.Marshal(&c)

	if err != nil {
		t.Fatalf("legacyMetric[int64].MarshalJSON %v: %v", c, err)
	}
	want := []byte(`3`)
	if !bytes.Equal(got, want) {
		t.Errorf("legacyMetric[int64].MarshalJSON: got %v, want %v", got, want)
	}
}

func TestLegacyFloatDPUnmarshalJSON(t *testing.T) {
	var got legacyMetric[float64]
	err := json.Unmarshal([]byte(`3.27`), &got)
	if err != nil {
		t.Fatalf("legacyMetric[float64].UnmarshalJSON: %v", err)
	}

	want := legacyMetric[float64]{3.27}

	if diff := cmp.Diff(want, got, cmp.AllowUnexported(legacyMetric[float64]{}, legacyMetric[int64]{})); diff != "" {
		t.Errorf("legacyMetric[float64].UnmarshalJSON:  diff (-want +got): %v", diff)
	}
}

func TestLegacyFloatDPMarshalJSON(t *testing.T) {
	f := legacyMetric[float64]{3.27}

	got, err := json.Marshal(&f)

	if err != nil {
		t.Fatalf("legacyMetric[float64].MarshalJSON %v: %v", f, err)
	}
	want := []byte(`3.27`)
	if !bytes.Equal(got, want) {
		t.Errorf("legacyMetric[float64].MarshalJSON: got %v, want %v", got, want)
	}
}
