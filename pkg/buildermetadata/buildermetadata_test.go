// Copyright 2025 Google LLC
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

package buildermetadata

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuilderMetadataUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
		want  BuilderMetadata
	}{
		{
			name:  "empty",
			input: []byte(`{}`),
			want: BuilderMetadata{
				map[MetadataID]MetadataValue{},
			},
		},
		{
			name:  "basic metadata",
			input: []byte(`{"m":{"1":"true","2":"false","3":"angular","4":"17.0.0","5":"@apphosting/adapter-angular","6":"17.2.3","7":"nx"}}`),
			want: BuilderMetadata{
				map[MetadataID]MetadataValue{"1": "true", "2": "false", "3": "angular", "4": "17.0.0", "5": "@apphosting/adapter-angular", "6": "17.2.3", "7": "nx"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got BuilderMetadata
			err := json.Unmarshal(tc.input, &got)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(BuilderMetadata{})); diff != "" {
				t.Errorf("BuilderMetadata json.Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuilderMetadataMarshalJSON(t *testing.T) {
	testCases := []struct {
		name  string
		input BuilderMetadata
		want  []byte
	}{
		{
			name:  "empty",
			input: BuilderMetadata{map[MetadataID]MetadataValue{}},
			want:  []byte(`{}`),
		},
		{
			name:  "basic metadata",
			input: BuilderMetadata{map[MetadataID]MetadataValue{"1": "true", "2": "false", "3": "angular", "4": "17.0.0", "5": "@apphosting/adapter-angular", "6": "17.2.3", "7": "nx"}},
			want:  []byte(`{"m":{"1":"true","2":"false","3":"angular","4":"17.0.0","5":"@apphosting/adapter-angular","6":"17.2.3","7":"nx"}}`),
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
