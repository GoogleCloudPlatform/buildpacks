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

	"github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktestenv"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/builderoutput"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpacks/libcnb/v2"
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

			ctx := NewContext()
			if ctx.debug != tc.want {
				t.Errorf("ctx.debug=%t, want %t", ctx.debug, tc.want)
			}
			if ctx.Debug() != tc.want {
				t.Errorf("ctx.Debug()=%t, want %t", ctx.debug, tc.want)
			}
		})
	}
}

func TestNewContextWithApplicationRoot(t *testing.T) {
	want := "myroot"
	got := NewContext(WithApplicationRoot(want)).applicationRoot
	if got != want {
		t.Errorf("NewContext().applicationRoot=%q want %q", got, want)
	}
}

func TestNewContextWithBuidpackInfo(t *testing.T) {
	want := libcnb.BuildpackInfo{Name: "myname"}
	got := NewContext(WithBuildpackInfo(want)).info
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewContext().info\ngot %#v\nwant %#v", got, want)
	}
}

func TestNewContextWithBuildContext(t *testing.T) {
	want := libcnb.BuildContext{StackID: "mystack"}
	got := NewContext(WithBuildContext(want)).buildContext
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewContext().buildContext\ngot %#v\nwant %#v", got, want)
	}
}

func TestDetectContextInitialized(t *testing.T) {
	setUpDetectEnvironment(t)

	id := "my-id"
	version := "my-version"
	name := "my-name"
	var ctx *Context
	detect(func(c *Context) (DetectResult, error) {
		ctx = c
		return OptIn("some reason"), nil
	}, libcnb.WithExitHandler(&fakeExitHandler{}))

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
	setUpDetectEnvironment(t)

	var ctx *Context
	detect(func(c *Context) (DetectResult, error) {
		ctx = c
		return OptIn("some reason"), nil
	}, libcnb.WithExitHandler(&fakeExitHandler{}))

	if len(ctx.stats.spans) != 1 {
		t.Errorf("len(spans)=%d want=1", len(ctx.stats.spans))
	}
	got := ctx.stats.spans[0]
	wantName := "Buildpack Detect"
	if !strings.HasPrefix(got.name, wantName) {
		t.Errorf("Unexpected span name got %q want prefix %q", got.name, wantName)
	}
	if got.start.IsZero() {
		t.Errorf("Start time not set")
	}
	if !got.end.After(got.start) {
		t.Errorf("End %v not after start %v", got.end, got.start)
	}
	if got.status != buildererror.StatusOk {
		t.Errorf("Unexpected status got=%s want=%s", got.status, buildererror.StatusOk)
	}
}

func TestDetectNilResult(t *testing.T) {
	setUpDetectEnvironment(t)

	handler := &fakeExitHandler{}
	// Tests that the function does not panic when both result and error are nil.
	detect(func(c *Context) (DetectResult, error) {
		return nil, nil
	}, libcnb.WithExitHandler(handler))

	// Tests that the function does not panic when both result and error are nil.
	if want, got := "detect did not return a result or an error", handler.err.Error(); !strings.Contains(got, want) {
		t.Errorf("ExitHandler.err = %q, should contain %q", got, want)
	}
}

func TestBuildContextInitialized(t *testing.T) {
	setUpBuildEnvironment(t)

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
	setUpBuildEnvironment(t)

	var ctx *Context
	build(func(c *Context) error {
		ctx = c
		return nil
	})

	if len(ctx.stats.spans) != 1 {
		t.Errorf("len(spans)=%d want=1", len(ctx.stats.spans))
	}
	got := ctx.stats.spans[0]
	wantName := "Buildpack Build"
	if !strings.HasPrefix(got.name, wantName) {
		t.Errorf("Unexpected span name got %q want prefix %q", got.name, wantName)
	}
	if got.start.IsZero() {
		t.Errorf("Start time not set")
	}
	if !got.end.After(got.start) {
		t.Errorf("End %v not after start %v", got.end, got.start)
	}
	if got.status != buildererror.StatusOk {
		t.Errorf("Unexpected status got=%s want=%s", got.status, buildererror.StatusOk)
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

	setUpBuildEnvironment(t)

	build(func(c *Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	fname := filepath.Join(tempDir, builderOutputFilename)
	var got builderoutput.BuilderOutput
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", fname, err)
	}
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if len(got.Stats) != 1 {
		t.Errorf("Incorrect length of stats, got %d, want %d", len(got.Stats), 1)
	}
	if got.Stats[0].DurationMs < 100 {
		t.Errorf("Duration is too short, got %d, want >= %d", got.Stats[0].DurationMs, 100)
	}
}

func TestAddWebProcess(t *testing.T) {
	ctx := NewContext()
	ctx.AddWebProcess([]string{"/start"})
	want := []libcnb.Process{proc("/start", "web")}

	if !reflect.DeepEqual(ctx.buildResult.Processes, want) {
		t.Errorf("Processes not equal got %#v, want %#v", ctx.buildResult.Processes, want)
	}
}

func TestAddProcess(t *testing.T) {
	testCases := []struct {
		desc    string
		name    string
		cmd     []string
		opts    []processOption
		initial []libcnb.Process
		want    []libcnb.Process
	}{
		{
			desc: "no args, no processes",
			name: "web",
			cmd:  []string{"/web"},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/web"}, Type: "web"},
			},
		},
		{
			desc: "add to existing",
			name: "web",
			cmd:  []string{"/web"},
			initial: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/dev"}, Type: "dev"},
				libcnb.Process{Command: []string{"bash", "-c", "/cli"}, Type: "cli"},
			},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/dev"}, Type: "dev"},
				libcnb.Process{Command: []string{"bash", "-c", "/cli"}, Type: "cli"},
				libcnb.Process{Command: []string{"bash", "-c", "/web"}, Type: "web"},
			},
		},
		{
			desc: "override existing",
			name: "web",
			cmd:  []string{"/OVERRIDE"},
			initial: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/dev"}, Type: "dev"},
				libcnb.Process{Command: []string{"bash", "-c", "/web"}, Type: "web"},
				libcnb.Process{Command: []string{"bash", "-c", "/cli"}, Type: "cli"},
			},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/dev"}, Type: "dev"},
				libcnb.Process{Command: []string{"bash", "-c", "/cli"}, Type: "cli"},
				libcnb.Process{Command: []string{"bash", "-c", "/OVERRIDE"}, Type: "web"},
			},
		},
		{
			desc: "no args",
			name: "foo",
			cmd:  []string{"/start"},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/start"}, Type: "foo"},
			},
		},
		{
			desc: "with args",
			name: "foo",
			cmd:  []string{"/start", "arg1", "arg2"},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/start arg1 arg2"}, Type: "foo"},
			},
		},
		{
			desc: "with opts, direct",
			name: "foo",
			cmd:  []string{"/start", "arg1", "arg2"},
			opts: []processOption{AsDirectProcess()},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"/start"}, Arguments: []string{"arg1", "arg2"}, Type: "foo"},
			},
		},
		{
			desc: "with opts, default",
			name: "foo",
			cmd:  []string{"/start", "arg1", "arg2"},
			opts: []processOption{AsDefaultProcess()},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"bash", "-c", "/start arg1 arg2"}, Type: "foo", Default: true},
			},
		},
		{
			desc: "with opts, direct default",
			name: "foo",
			cmd:  []string{"/start", "arg1", "arg2"},
			opts: []processOption{AsDirectProcess(), AsDefaultProcess()},
			want: []libcnb.Process{
				libcnb.Process{Command: []string{"/start"}, Arguments: []string{"arg1", "arg2"}, Type: "foo", Default: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := NewContext()
			ctx.buildResult.Processes = tc.initial

			ctx.AddProcess(tc.name, tc.cmd, tc.opts...)

			if !reflect.DeepEqual(ctx.buildResult.Processes, tc.want) {
				t.Errorf("Processes not equal got %#v, want %#v", ctx.buildResult.Processes, tc.want)
			}
		})
	}
}

func TestAddLabel(t *testing.T) {
	testCases := []struct {
		name      string
		keyvalues []string
		value     string
		want      []libcnb.Label
	}{
		{
			name:      "simple",
			keyvalues: []string{"my-key=my-value"},
			want:      []libcnb.Label{{Key: "google.my-key", Value: "my-value"}},
		},
		{
			name:      "uppercase key",
			keyvalues: []string{"MY-KEY=my-value"},
			want:      []libcnb.Label{{Key: "google.my-key", Value: "my-value"}},
		},
		{
			name:      "mixed case value",
			keyvalues: []string{"my-key=My-Value"},
			want:      []libcnb.Label{{Key: "google.my-key", Value: "My-Value"}},
		},
		{
			name:      "underscore to dash key",
			keyvalues: []string{"my_key=My-Value"},
			want:      []libcnb.Label{{Key: "google.my-key", Value: "My-Value"}},
		},
		{
			name:      "multiple",
			keyvalues: []string{"my-key=My-Value", "my-other-key=my-other-value"},
			want: []libcnb.Label{
				{Key: "google.my-key", Value: "My-Value"},
				{Key: "google.my-other-key", Value: "my-other-value"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewContext()

			for _, kv := range tc.keyvalues {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					t.Fatalf("incorrect format %q, expect key=value", kv)
				}
				ctx.AddLabel(parts[0], parts[1])
			}

			if !reflect.DeepEqual(ctx.buildResult.Labels, tc.want) {
				t.Errorf("Labels not equal got %#v, want %#v", ctx.buildResult.Labels, tc.want)
			}
		})
	}
}

func TestAddLabelErrors(t *testing.T) {
	invalids := []string{"", "0", "00invalid", "abc def", "abd@def", "  abc", "def  ", "a__b"}

	for _, invalid := range invalids {
		ctx := NewContext()
		ctx.AddLabel(invalid, "some-value")

		if len(ctx.buildResult.Labels) > 0 {
			t.Errorf("invalid label %q was incorrectly included", invalid)
		}
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
			dir, cleanup := buildpacktestenv.TempWorkingDir(t)
			defer cleanup()

			ctx := NewContext(WithApplicationRoot(dir))
			for _, f := range tc.files {
				if err := os.MkdirAll(tc.prefix, 0777); err != nil {
					t.Fatalf("Error creating %s: %v", tc.prefix, err)
				}
				_, err := ioutil.TempFile(tc.prefix, f)
				if err != nil {
					t.Fatalf("Creating temp file %s/%s: %v", tc.prefix, f, err)
				}
			}

			pattern := "*.py"
			got, err := ctx.HasAtLeastOne(pattern)
			if err != nil {
				t.Errorf("HasAtLeastOne(%v) failed unexpectedly; err=%s", pattern, err)
			}
			if got != tc.want {
				t.Errorf("HasAtLeastOne(%v)=%t, want=%t", pattern, got, tc.want)
			}
		})
	}
}

func TestHasAtLeastOneFiltered(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
		files  []string
		filter filepathFilter
		want   bool
	}{
		{
			name:   "empty",
			prefix: ".",
			files:  []string{},
			filter: nil,
			want:   false,
		},
		{
			name:   "single_file_nil_filter",
			prefix: ".",
			files:  []string{"*.py"},
			want:   true,
		},
		{
			name:   "single_file_wrong_name",
			prefix: ".",
			files:  []string{"*.rb"},
			filter: nil,
			want:   false,
		},
		{
			name:   "multiple_files_nil_filter",
			prefix: ".",
			files:  []string{"*.py", "*.rb"},
			filter: nil,
			want:   true,
		},
		{
			name:   "subfolder_contains_file",
			prefix: "sub",
			files:  []string{"*.py"},
			filter: nil,
			want:   true,
		},
		{
			name:   "subfolder_contains_wrong_name",
			prefix: "sub",
			files:  []string{"*.rb"},
			filter: nil,
			want:   false,
		},
		{
			name:   "subfolder_respects_false_filter",
			prefix: "node_modules",
			files:  []string{"*.py"},
			filter: func(path string) bool {
				return false
			},
			want: false,
		},
		{
			name:   "subfolder_respects_true_filter",
			prefix: "node_modules",
			files:  []string{"*.py"},
			filter: func(path string) bool {
				return true
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := buildpacktestenv.TempWorkingDir(t)
			defer cleanup()

			ctx := NewContext(WithApplicationRoot(dir))
			for _, f := range tc.files {
				prefixDir := filepath.Join(dir, tc.prefix)
				if err := os.MkdirAll(prefixDir, 0777); err != nil {
					t.Fatalf("Error creating %s: %v", prefixDir, err)
				}
				_, err := ioutil.TempFile(prefixDir, f)
				if err != nil {
					t.Fatalf("Creating temp file %s/%s: %v", prefixDir, f, err)
				}
			}

			pattern := "*.py"
			got, err := ctx.HasAtLeastOneFiltered(pattern, tc.filter)
			if err != nil {
				t.Errorf("HasAtLeastOneFiltered(%v) failed unexpectedly; err=%s", pattern, err)
			}
			if got != tc.want {
				t.Errorf("HasAtLeastOneFiltered(%v)=%t, want=%t", pattern, got, tc.want)
			}
		})
	}
}

func TestHasAtLeastOneOutsideDependencyDirectories(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
		files  []string
		want   bool
	}{
		{
			name:   "detects_file_in_root",
			prefix: ".",
			files:  []string{"*.py"},
			want:   true,
		},
		{
			name:   "ignores_file_in_node_modules",
			prefix: "node_modules",
			files:  []string{"*.py"},
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, cleanup := buildpacktestenv.TempWorkingDir(t)
			defer cleanup()

			ctx := NewContext(WithApplicationRoot(dir))
			for _, f := range tc.files {
				prefixDir := filepath.Join(dir, tc.prefix)
				if err := os.MkdirAll(prefixDir, 0777); err != nil {
					t.Fatalf("Error creating %s: %v", prefixDir, err)
				}
				_, err := ioutil.TempFile(prefixDir, f)
				if err != nil {
					t.Fatalf("Creating temp file %s/%s: %v", prefixDir, f, err)
				}
			}

			pattern := "*.py"
			got, err := ctx.HasAtLeastOneOutsideDependencyDirectories(pattern)
			if err != nil {
				t.Errorf("HasAtLeastOneOutsideDependencyDirectories(%v) failed unexpectedly; err=%s", pattern, err)
			}
			if got != tc.want {
				t.Errorf("HasAtLeastOneOutsideDependencyDirectories(%v)=%t, want=%t", pattern, got, tc.want)
			}
		})
	}
}

func proc(command, commandType string) libcnb.Process {
	return libcnb.Process{Command: []string{command}, Type: commandType, Default: true}
}

// fakeExitHandler allows libcnb's Detect() function to be called without causing an os.Exit().
type fakeExitHandler struct {
	err        error
	errCalled  bool
	passCalled bool
	failCalled bool
}

// Error is called when an error is encountered.
func (eh *fakeExitHandler) Error(err error) {
	eh.errCalled = true
	eh.err = err
}

// Fail is called when a buildpack fails.
func (eh *fakeExitHandler) Fail() {
	eh.failCalled = true
}

// Pass is called when a buildpack passes.
func (eh *fakeExitHandler) Pass() {
	eh.passCalled = true
}

func simpleContext(t *testing.T) (*Context, func()) {
	t.Helper()
	setUpDetectEnvironment(t)
	c := NewContext()
	// simpleContext relies on t.Cleanup() for cleanup now and no longer
	// has to return a cleanup func, but calling sites expect a cleanup func.
	return c, func() {}
}

// setUpDetectEnvironment sets up an environment for testing buildpack detect
// functionality.
func setUpDetectEnvironment(t *testing.T) buildpacktestenv.TempDirs {
	t.Helper()
	temps := buildpacktestenv.SetUpTempDirs(t, "")
	setOSArgs(t, []string{filepath.Join(temps.BuildpackDir, "bin", "detect"), temps.PlatformDir, temps.PlanFile})

	return temps
}

// setUpBuildEnvironment sets up an environment for testing buildpack build
// functionality.
func setUpBuildEnvironment(t *testing.T) buildpacktestenv.TempDirs {
	t.Helper()
	temps := buildpacktestenv.SetUpTempDirs(t, "")
	setOSArgs(t, []string{filepath.Join(temps.BuildpackDir, "bin", "build"), temps.LayersDir, temps.PlatformDir, temps.PlanFile})

	return temps
}

func setOSArgs(t *testing.T, args []string) {
	t.Helper()
	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}
