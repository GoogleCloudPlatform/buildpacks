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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/builderoutput"

	"github.com/buildpacks/libcnb"
	"github.com/google/go-cmp/cmp"
)

func TestSaveErrorOutput(t *testing.T) {
	t.Cleanup(buildermetrics.Reset)
	tempDir, err := os.MkdirTemp("", "save-error-output-")
	if err != nil {
		t.Fatalf("creating temp dir: %v", err)
	}

	os.Setenv("BUILDER_OUTPUT", tempDir)
	defer func() {
		os.Unsetenv("BUILDER_OUTPUT")
	}()

	oldMax := maxMessageBytes
	maxMessageBytes = 8
	defer func() {
		maxMessageBytes = oldMax
	}()
	ctx := NewContext(WithBuildpackInfo(libcnb.BuildpackInfo{ID: "id", Version: "version"}))
	msg := "This is a long message that will be truncated."

	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)

	ctx.saveErrorOutput(buildererror.Errorf(buildererror.StatusInternal, msg))

	data, err := os.ReadFile(filepath.Join(tempDir, "output"))
	if err != nil {
		t.Fatalf("TestSaveErrorOutput reading expected file $BUILDER_OUTPUT/output: %v", err)
	}
	var got builderoutput.BuilderOutput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("TestSaveErrorOutput unmarshal json: %v", err)
	}

	bm := buildermetrics.NewBuilderMetrics()
	bm.GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)
	want := builderoutput.BuilderOutput{
		Metrics: bm,
		Error: buildererror.Error{
			BuildpackID:      "id",
			BuildpackVersion: "version",
			Type:             buildererror.StatusInternal,
			Status:           buildererror.StatusInternal,
			ID:               buildererror.GenerateErrorID(msg),
			Message:          "...ated.",
		},
	}

	if !cmp.Equal(got, want, cmp.AllowUnexported(buildermetrics.BuilderMetrics{}, buildermetrics.Counter{}, buildererror.Error{})) {
		t.Errorf("TestSaveErrorOutput expected output does not match\ngot:\n%#v\nwant:\n%#v", got, want)
	}
}

func TestMessageProducers(t *testing.T) {
	testCases := []struct {
		name     string
		producer MessageProducer
		stdout   string
		stderr   string
		combined string
		want     string
	}{
		{
			name:     "KeepStdoutTail",
			producer: KeepStdoutTail,
			stdout:   "123456789stdout",
			want:     "...stdout",
		},
		{
			name:     "KeepStderrTail",
			producer: KeepStderrTail,
			stderr:   "123456789stderr",
			want:     "...stderr",
		},
		{
			name:     "KeepCombinedTail",
			producer: KeepCombinedTail,
			combined: "123456789combined",
			want:     "...mbined",
		},
		{
			name:     "KeepStdoutHead",
			producer: KeepStdoutHead,
			stdout:   "stdout123456789",
			want:     "stdout...",
		},
		{
			name:     "KeepStderrHead",
			producer: KeepStderrHead,
			stderr:   "stderr123456789",
			want:     "stderr...",
		},
		{
			name:     "KeepCombinedHead",
			producer: KeepCombinedHead,
			combined: "combined123456789",
			want:     "combin...",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldMax := maxMessageBytes
			maxMessageBytes = 9
			defer func() {
				maxMessageBytes = oldMax
			}()

			got := tc.producer(&ExecResult{Stdout: tc.stdout, Stderr: tc.stderr, Combined: tc.combined})

			if got != tc.want {
				t.Errorf("message got %q want %q", got, tc.want)
			}
		})
	}
}

func TestKeepTail(t *testing.T) {
	testCases := []struct {
		name            string
		message         string
		want            string
		maxMessageBytes int
	}{
		{
			name:            "empty message",
			message:         "",
			want:            "",
			maxMessageBytes: 8,
		},
		{
			name:            "short message",
			message:         "123",
			want:            "123",
			maxMessageBytes: 8,
		},
		{
			name:            "long message",
			message:         "12345678901234567890",
			want:            "...67890",
			maxMessageBytes: 8,
		},
		{
			name:            "boundary message 1",
			message:         "12345678",
			want:            "12345678",
			maxMessageBytes: 8,
		},
		{
			name:            "boundary message 2",
			message:         "123456789",
			want:            "...56789",
			maxMessageBytes: 8,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.maxMessageBytes != 0 {
				oldMax := maxMessageBytes
				maxMessageBytes = tc.maxMessageBytes
				defer func() {
					maxMessageBytes = oldMax
				}()
			}
			got := keepTail(tc.message)
			if got != tc.want {
				t.Errorf("keepTail() got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestKeepHead(t *testing.T) {
	testCases := []struct {
		name            string
		message         string
		want            string
		maxMessageBytes int
	}{
		{
			name:            "empty message",
			message:         "",
			want:            "",
			maxMessageBytes: 8,
		},
		{
			name:            "short message",
			message:         "123",
			want:            "123",
			maxMessageBytes: 8,
		},
		{
			name:            "long message",
			message:         "12345678901234567890",
			want:            "12345...",
			maxMessageBytes: 8,
		},
		{
			name:            "boundary message 1",
			message:         "12345678",
			want:            "12345678",
			maxMessageBytes: 8,
		},
		{
			name:            "boundary message 2",
			message:         "123456789",
			want:            "12345...",
			maxMessageBytes: 8,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.maxMessageBytes != 0 {
				oldMax := maxMessageBytes
				maxMessageBytes = tc.maxMessageBytes
				defer func() {
					maxMessageBytes = oldMax
				}()
			}
			got := keepHead(tc.message)
			if got != tc.want {
				t.Errorf("keepHead() got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestSaveBuilderSuccessOutput(t *testing.T) {
	dur := 30 * time.Second
	userDur := 5 * time.Second
	buildpackID, buildpackVersion := "my-id", "my-version"
	metrics3 := buildermetrics.NewBuilderMetrics()
	metrics3.GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)

	metrics6 := buildermetrics.NewBuilderMetrics()
	metrics6.GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(6)

	testCases := []struct {
		name                     string
		addMetrics               bool
		installedRuntimeVersions []string
		initial                  *builderoutput.BuilderOutput
		warnings                 []string
		want                     builderoutput.BuilderOutput
	}{
		{
			name:       "no file",
			addMetrics: true, // adds 3
			want: builderoutput.BuilderOutput{
				Metrics: metrics3,
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
			},
		},
		{
			name:     "no file warnings",
			warnings: []string{"Test warning about a conflicting file."},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{"Test warning about a conflicting file."},
			},
		},
		{
			name: "existing file",
			initial: &builderoutput.BuilderOutput{
				InstalledRuntimeVersions: []string{"1.0.0"},
				Metrics:                  metrics3,
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
				},
			},
			installedRuntimeVersions: []string{"3.2.1", "6.0.3"},
			addMetrics:               true, // adds 3
			want: builderoutput.BuilderOutput{
				InstalledRuntimeVersions: []string{"1.0.0", "3.2.1", "6.0.3"},
				Metrics:                  metrics6,
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				CustomImage: false,
			},
		},
		{
			name:                     "propagates InstalledRuntimeVersions",
			installedRuntimeVersions: []string{"3.2.1", "6.0.3"},
			want: builderoutput.BuilderOutput{
				InstalledRuntimeVersions: []string{"3.2.1", "6.0.3"},
				Metrics:                  buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				CustomImage: false,
			},
		},
		{
			name: "existing file new warnings",
			initial: &builderoutput.BuilderOutput{
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
				},
			},
			warnings: []string{"Test warning about a conflicting file."},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings:    []string{"Test warning about a conflicting file."},
				CustomImage: false,
			},
		},
		{
			name: "existing file existing warnings",
			initial: &builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
				},
				Warnings: []string{"Test warning from a previous buildpack."},
			},
			warnings: []string{"Test warning about a conflicting file."},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: "bp1", BuildpackVersion: "v1", DurationMs: 1000, UserDurationMs: 100},
					{BuildpackID: "bp2", BuildpackVersion: "v2", DurationMs: 2000, UserDurationMs: 200},
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{
					"Test warning from a previous buildpack.",
					"Test warning about a conflicting file.",
				},
				CustomImage: false,
			},
		},
		{
			name: "warnings trim last",
			warnings: []string{
				"Test warning about a conflicting file.",
				strings.Repeat("x", maxMessageBytes),
			},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{
					"Test warning about a conflicting file.",
					strings.Repeat("x", 48676) + "...",
				},
				CustomImage: false,
			},
		},
		{
			name: "warnings trim last short",
			warnings: []string{"Test warning about a conflicting file.",
				strings.Repeat("x", 48676-4), // Four bytes shorter than the maximum which should leave exactly one character for the second warning.
				strings.Repeat("y", maxMessageBytes),
			},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{
					"Test warning about a conflicting file.",
					strings.Repeat("x", 48672),
					"y...",
				},
				CustomImage: false,
			},
		},
		{
			name: "warnings drop last short",
			warnings: []string{"Test warning about a conflicting file.",
				strings.Repeat("x", 48676-3), // Three bytes shorter than the maximum, which would leave 3 characters for the last warning so we drop it.
				strings.Repeat("y", maxMessageBytes),
			},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{
					"Test warning about a conflicting file.",
					strings.Repeat("x", 48673),
				},
				CustomImage: false,
			},
		},
		{
			name: "warnings drop last and trim",
			warnings: []string{"Test warning about a conflicting file.",
				strings.Repeat("x", maxMessageBytes),
				strings.Repeat("y", maxMessageBytes),
			},
			want: builderoutput.BuilderOutput{
				Metrics: buildermetrics.NewBuilderMetrics(),
				Stats: []builderoutput.BuilderStat{
					{BuildpackID: buildpackID, BuildpackVersion: buildpackVersion, DurationMs: dur.Milliseconds(), UserDurationMs: userDur.Milliseconds()},
				},
				Warnings: []string{
					"Test warning about a conflicting file.",
					strings.Repeat("x", 48676) + "...",
				},
				CustomImage: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(buildermetrics.Reset)
			tempDir, err := os.MkdirTemp("", "save-success-output-")
			if err != nil {
				t.Fatalf("TestSaveBuilderSuccessOutput creating temp dir: %v", err)
			}

			os.Setenv("BUILDER_OUTPUT", tempDir)
			defer func() {
				os.Unsetenv("BUILDER_OUTPUT")
			}()

			fname := filepath.Join(tempDir, builderOutputFilename)
			if tc.initial != nil {
				content, err := json.Marshal(tc.initial)
				if err != nil {
					t.Fatalf("Failed to marshal stats: %v", err)
				}
				if err := os.WriteFile(fname, content, 0644); err != nil {
					t.Fatalf("TestSaveBuilderSuccessOutput writing %s: %v", fname, err)
				}
			}
			ctx := NewContext(WithBuildpackInfo(libcnb.BuildpackInfo{ID: buildpackID, Version: buildpackVersion}))

			if tc.addMetrics {
				buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)
			}
			ctx.stats.user = userDur
			ctx.warnings = tc.warnings
			for _, version := range tc.installedRuntimeVersions {
				ctx.AddInstalledRuntimeVersion(version)
			}

			ctx.saveSuccessOutput(dur)

			content, err := os.ReadFile(fname)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", fname, err)
			}
			got, err := builderoutput.FromJSON(content)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if !cmp.Equal(got, tc.want, cmp.AllowUnexported(buildermetrics.BuilderMetrics{}, buildermetrics.Counter{}, buildererror.Error{})) {
				t.Errorf("%v: Expected stats do not match got %#v, want %#v", tc.name, got, tc.want)
				t.Errorf("saveBuilderSuccessOutput metrics proto: got: %v, want: %v", got.Metrics, tc.want.Metrics)
			}
		})
	}
}
