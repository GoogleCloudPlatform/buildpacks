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

package main

import (
	"os"
	"strconv"
	"strings"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		stack string
		want  int
	}{
		{
			name: "with target",
			env:  []string{"GOOGLE_FUNCTION_TARGET=helloWorld"},
			want: 0,
		},
		{
			name: "without target",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestGetMaxOldSpaceSize(t *testing.T) {
	testCases := []struct {
		name    string
		env     []string
		want    int
		wantErr bool
	}{
		{
			name: "env val not set",
		},
		{
			name:    "env val set but no value",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB="},
			wantErr: true,
		},
		{
			name:    "env val set but less than head room",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=" + strconv.Itoa(nodeJSHeadroomMB-1)},
			wantErr: true,
		},
		{
			name:    "env val set but 0",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=0"},
			wantErr: true,
		},
		{
			name:    "env val set but negative",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=-10"},
			wantErr: true,
		},
		{
			name:    "env val set not integer",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=1a2b"},
			wantErr: true,
		},
		{
			name:    "env val set but equal to head room",
			env:     []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=" + strconv.Itoa(nodeJSHeadroomMB)},
			wantErr: true,
		},
		{
			name: "env val set and greater than head room",
			env:  []string{"GOOGLE_CONTAINER_MEMORY_HINT_MB=4096"},
			want: 4096 - nodeJSHeadroomMB,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, keyVal := range tc.env {
				setEnv(t, keyVal)
			}

			got, err := getMaxOldSpaceSize()
			gotErr := err != nil

			if gotErr != tc.wantErr {
				t.Errorf("getMaxOldSpaceSize() got err=%t, want err=%t. err: %v", gotErr, tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("getMaxOldSpaceSize()=%d, want=%d", got, tc.want)
			}
		})
	}
}

func setEnv(t *testing.T, keyVal string) {
	t.Helper()

	kv := strings.SplitN(keyVal, "=", 2)
	if len(kv) != 2 {
		t.Fatal("Want env var in form KEY=VALUE")
	}

	old, oldPresent := os.LookupEnv(kv[0])
	if err := os.Setenv(kv[0], kv[1]); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if oldPresent {
			if err := os.Setenv(kv[0], old); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Unsetenv(kv[0]); err != nil {
			t.Fatal(err)
		}
	})
}
