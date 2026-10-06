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

package acceptance

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStructureTestJSON(t *testing.T) {
	testCases := []struct {
		name   string
		checks StructureTest
		want   string
	}{
		{
			name:   "empty configuration",
			checks: StructureTest{},
			want:   `{"SchemaVersion":"","MetadataTest":{"EnvVars":null,"ExposedPorts":null,"Entrypoint":null,"Cmd":null,"Workdir":""},"FileExistenceTests":null}`,
		},
		{
			name: "check empty cmd",
			checks: StructureTest{
				MetadataTest: metadataTest{
					Cmd: []string{},
				},
			},
			want: `{"SchemaVersion":"","MetadataTest":{"EnvVars":null,"ExposedPorts":null,"Entrypoint":null,"Cmd":[],"Workdir":""},"FileExistenceTests":null}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			json, err := json.Marshal(tc.checks)

			if err != nil {
				t.Errorf("Couldn't convert +#v to json: %v", err)
			}
			if string(json) != tc.want {
				t.Errorf("Invalid json, got=%s, want=%s", json, tc.want)
			}
		})
	}
}

func TestNewStructureTest(t *testing.T) {
	testCases := []struct {
		name              string
		filesMustExist    []string
		filesMustNotExist []string
		want              *StructureTest
	}{
		{
			name:              "no check",
			filesMustExist:    nil,
			filesMustNotExist: nil,
			want:              nil,
		},
		{
			name:              "check files",
			filesMustExist:    []string{"/bin/name"},
			filesMustNotExist: []string{"/notfound"},
			want: &StructureTest{
				SchemaVersion: "2.0.0",
				FileExistenceTests: []fileExistenceTest{
					{
						Name:        "/bin/name",
						Path:        "/bin/name",
						ShouldExist: true,
						UID:         -1,
						GID:         -1,
					},
					{
						Name:        "/notfound",
						Path:        "/notfound",
						ShouldExist: false,
						UID:         -1,
						GID:         -1,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			st := NewStructureTest(tc.filesMustExist, tc.filesMustNotExist)

			if !reflect.DeepEqual(st, tc.want) {
				t.Errorf("NewStructureTest() got=%#v, want=%#v", st, tc.want)
			}
		})
	}
}
