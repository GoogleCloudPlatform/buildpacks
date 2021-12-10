// Copyright 2020 Google LLC
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

package buildererror

import (
	"bytes"
	"reflect"
	"testing"
)

func TestUnmarshalJSON(t *testing.T) {
	var s Status
	err := s.UnmarshalJSON([]byte(`"PERMISSION_DENIED"`))
	if err != nil {
		t.Fatal(err)
	}

	want := StatusPermissionDenied

	if !reflect.DeepEqual(s, want) {
		t.Errorf("status parsing failed got: %v, want: %v", s, want)
	}
}

func TestMarshalJSON(t *testing.T) {
	s := StatusResourceExhausted

	j, err := s.MarshalJSON()

	if err != nil {
		t.Fatalf("Failed to marshal %v: %v", s, err)
	}
	want := []byte(`"RESOURCE_EXHAUSTED"`)
	if !bytes.Equal(want, j) {
		t.Errorf("got %v, want %v", j, want)
	}
}
