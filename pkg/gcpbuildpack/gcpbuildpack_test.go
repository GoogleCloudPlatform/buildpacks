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

package gcpbuildpack

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpack/libbuildpack/buildpack"
	"github.com/buildpack/libbuildpack/layers"
)

func TestDebugModeInitialized(t *testing.T) {
	testCases := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name: "no env var",
		},
		{
			name:  "true env var",
			value: "true",
			want:  true,
		},
		{
			name:  "false env var",
			value: "false",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				if err := os.Setenv(env.DebugMode, tc.value); err != nil {
					t.Fatalf("Failed to set env: %v", err)
				}
				defer func() {
					if err := os.Unsetenv(env.DebugMode); err != nil {
						t.Fatalf("Failed to unset env: %v", err)
					}
				}()
			} else {
				if err := os.Unsetenv(env.DebugMode); err != nil {
					t.Fatalf("Failed to unset env: %v", err)
				}
			}

			ctx := NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
			if ctx.debug != tc.want {
				t.Errorf("ctx.debug=%t, want %t", ctx.debug, tc.want)
			}
			if ctx.Debug() != tc.want {
				t.Errorf("ctx.Debug()=%t, want %t", ctx.debug, tc.want)
			}
		})
	}
}

func TestDetectContextInitialized(t *testing.T) {
	_, cleanUp := setUpDetectEnvironment(t)
	defer cleanUp()

	id := "my-id"
	version := "my-version"
	name := "my-name"

	var ctx *Context
	detect(func(c *Context) error {
		ctx = c
		return nil
	})

	if ctx.BuildpackID() != id {
		t.Errorf("Unexpected id got=%q want=%q", ctx.BuildpackID(), id)
	}
	if ctx.BuildpackVersion() != version {
		t.Errorf("Unexpected version got=%q want=%q", ctx.BuildpackVersion(), version)
	}
	if ctx.BuildpackName() != name {
		t.Errorf("Unexpected name got=%q want=%q", ctx.BuildpackName(), name)
	}
}

func TestDetectEmitsSpan(t *testing.T) {
	_, cleanUp := setUpDetectEnvironment(t)
	defer cleanUp()

	var ctx *Context
	detect(func(c *Context) error {
		ctx = c
		return nil
	})

	if len(ctx.stats.spans) != 1 {
		t.Fatalf("len(spans)=%d want=1", len(ctx.stats.spans))
	}
	got := ctx.stats.spans[0]
	wantName := "Buildpack Detect"
	if !strings.HasPrefix(got.name, wantName) {
		t.Errorf("Unexpected span name got %q want prefix %q", got.name, wantName)
	}
	if got.start.IsZero() {
		t.Error("Start time not set")
	}
	if !got.end.After(got.start) {
		t.Errorf("End %v not after start %v", got.end, got.start)
	}
	if got.status != StatusOk {
		t.Errorf("Unexpected status got=%s want=%s", got.status, StatusOk)
	}
}

// func TestDetectCallbackReturingErrorExits(t *testing.T) {}
// func TestDetectFinalizes(t *testing.T) {}

func TestBuildContextInitialized(t *testing.T) {
	_, cleanUp := setUpBuildEnvironment(t)
	defer cleanUp()

	id := "my-id"
	version := "my-version"
	name := "my-name"

	var ctx *Context
	build(func(c *Context) error {
		ctx = c
		return nil
	})

	if ctx.BuildpackID() != id {
		t.Errorf("Unexpected id got=%q want=%q", ctx.BuildpackID(), id)
	}
	if ctx.BuildpackVersion() != version {
		t.Errorf("Unexpected version got=%q want=%q", ctx.BuildpackVersion(), version)
	}
	if ctx.BuildpackName() != name {
		t.Errorf("Unexpected name got=%q want=%q", ctx.BuildpackName(), name)
	}
}

func TestBuildEmitsSpan(t *testing.T) {
	_, cleanUp := setUpBuildEnvironment(t)
	defer cleanUp()

	var ctx *Context
	build(func(c *Context) error {
		ctx = c
		return nil
	})

	if len(ctx.stats.spans) != 1 {
		t.Fatalf("len(spans)=%d want=1", len(ctx.stats.spans))
	}
	got := ctx.stats.spans[0]
	wantName := "Buildpack Build"
	if !strings.HasPrefix(got.name, wantName) {
		t.Errorf("Unexpected span name got %q want prefix %q", got.name, wantName)
	}
	if got.start.IsZero() {
		t.Error("Start time not set")
	}
	if !got.end.After(got.start) {
		t.Errorf("End %v not after start %v", got.end, got.start)
	}
	if got.status != StatusOk {
		t.Errorf("Unexpected status got=%s want=%s", got.status, StatusOk)
	}
}

func TestBuildEmitsSuccessOutput(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "build-emits-success-output-")
	if err != nil {
		t.Fatalf("Creating temp dir: %v", err)
	}

	os.Setenv("BUILDER_OUTPUT", tempDir)
	defer func() {
		os.Unsetenv("BUILDER_OUTPUT")
	}()

	_, cleanUp := setUpBuildEnvironment(t)
	defer cleanUp()

	build(func(c *Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	fname := filepath.Join(tempDir, builderOutputFilename)
	var got builderOutput
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", fname, err)
	}
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(got.Stats) != 1 {
		t.Fatalf("Incorrect length of stats, got %d, want %d", len(got.Stats), 1)
	}
	if got.Stats[0].DurationMs < 100 {
		t.Errorf("Duration is too short, got %d, want >= %d", got.Stats[0].DurationMs, 100)
	}
}

func TestAddWebProcess(t *testing.T) {
	testCases := []struct {
		name    string
		initial layers.Processes
		cmd     []string
		want    layers.Processes
	}{
		{
			name:    "empty processes",
			initial: layers.Processes{},
			cmd:     []string{"/web"},
			want:    layers.Processes{proc("/web", "web")},
		},
		{
			name:    "existing web",
			initial: layers.Processes{proc("/dev", "dev"), proc("/web", "web"), proc("/cli", "cli")},
			cmd:     []string{"/OVERRIDE"},
			want:    layers.Processes{proc("/dev", "dev"), proc("/cli", "cli"), proc("/OVERRIDE", "web")},
		},
		{
			name:    "no web",
			initial: layers.Processes{proc("/dev", "dev"), proc("/cli", "cli")},
			cmd:     []string{"/web"},
			want:    layers.Processes{proc("/dev", "dev"), proc("/cli", "cli"), proc("/web", "web")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
			ctx.processes = tc.initial

			ctx.AddWebProcess(tc.cmd)

			if !reflect.DeepEqual(ctx.processes, tc.want) {
				t.Errorf("Processes not equal got %#v, want %#v", ctx.processes, tc.want)
			}
		})
	}
}

func TestHasAtLeastOne(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
		files  []string
		want   bool
	}{
		{
			name:   "empty",
			prefix: ".",
			files:  []string{},
			want:   false,
		},
		{
			name:   "single_file",
			prefix: ".",
			files:  []string{"*.py"},
			want:   true,
		},
		{
			name:   "single_file_wrong_name",
			prefix: ".",
			files:  []string{"*.rb"},
			want:   false,
		},
		{
			name:   "multiple_files",
			prefix: ".",
			files:  []string{"*.py", "*.rb"},
			want:   true,
		},
		{
			name:   "subfolder_contains_file",
			prefix: "sub",
			files:  []string{"*.py"},
			want:   true,
		},
		{
			name:   "subfolder_contains_wrong_name",
			prefix: "sub",
			files:  []string{"*.rb"},
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := tempWorkingDir(t)
			defer cleanup()

			ctx := NewContextForTests(buildpack.Info{ID: "id", Version: "version", Name: "name"}, dir)
			for _, f := range tc.files {
				ctx.MkdirAll(tc.prefix, 0777)
				_, err := ioutil.TempFile(tc.prefix, f)
				if err != nil {
					t.Fatalf("Creating temp file %s/%s: %v", tc.prefix, f, err)
				}
			}

			got := ctx.HasAtLeastOne("*.py")
			if got != tc.want {
				t.Errorf("HasAtLeastOne()=%t, want=%t", got, tc.want)
			}
		})
	}
}

func proc(command, commandType string) layers.Process {
	return layers.Process{Command: command, Type: commandType, Direct: true}
}
