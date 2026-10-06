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

package builderoutput

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	"github.com/google/go-cmp/cmp"
)

func TestFromJSON(t *testing.T) {
	serialized := `
{
	"rtVersions": ["6.0.6"],
  "metrics": {"c":{"1":3},"f":{"9":18.3}},
	"error": {
		"buildpackId": "bad-buildpack",
		"buildpackVersion": "vbad",
		"errorType": "INTERNAL",
		"canonicalCode": "INTERNAL",
		"errorId": "abc123",
		"errorMessage": "error-message",
		"anotherThing": 123
	},
	"metadata": {"m":{"1":"true", "2":"false", "3":"angular", "4":"17.0.0", "5":"@apphosting/adapter-angular", "6":"17.2.3", "7":"nx", "8":"uv", "9":"pyproject.toml"}},
	"stats": [
		{
			"buildpackId": "buildpack-1",
			"buildpackVersion": "v1",
			"totalDurationMs": 100,
			"userDurationMs": 101,
			"anotherThing": "shouldn't cause a problem"
		},
		{
			"buildpackId": "buildpack-2",
			"buildpackVersion": "v2",
			"totalDurationMs": 200,
			"userDurationMs": 201
		}
	],
	"warnings": [
		"Some warning",
		"Some other warning"
	],
	"customImage": true
}
`

	got, err := FromJSON([]byte(serialized))
	if err != nil {
		t.Fatal(err)
	}

	bm := buildermetrics.NewBuilderMetrics()
	bm.GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)
	bm.GetFloatDP(buildermetrics.ComposerInstallLatencyID).Add(18.3)
	fm := buildermetadata.NewBuilderMetadata()
	fm.SetValue(buildermetadata.IsUsingGenkit, "true")
	fm.SetValue(buildermetadata.IsUsingGenAI, "false")
	fm.SetValue(buildermetadata.FrameworkName, "angular")
	fm.SetValue(buildermetadata.FrameworkVersion, "17.0.0")
	fm.SetValue(buildermetadata.AdapterName, "@apphosting/adapter-angular")
	fm.SetValue(buildermetadata.AdapterVersion, "17.2.3")
	fm.SetValue(buildermetadata.MonorepoName, "nx")
	fm.SetValue(buildermetadata.PackageManager, "uv")
	fm.SetValue(buildermetadata.ConfigFile, "pyproject.toml")
	want := BuilderOutput{
		InstalledRuntimeVersions: []string{"6.0.6"},
		Metrics:                  bm,
		Error: buildererror.Error{
			BuildpackID:      "bad-buildpack",
			BuildpackVersion: "vbad",
			Type:             buildererror.StatusInternal,
			Status:           buildererror.StatusInternal,
			ID:               "abc123",
			Message:          "error-message",
		},
		Metadata: fm,
		Stats: []BuilderStat{
			{
				BuildpackID:      "buildpack-1",
				BuildpackVersion: "v1",
				DurationMs:       100,
				UserDurationMs:   101,
			},
			{
				BuildpackID:      "buildpack-2",
				BuildpackVersion: "v2",
				DurationMs:       200,
				UserDurationMs:   201,
			},
		},
		Warnings: []string{
			"Some warning",
			"Some other warning",
		},
		CustomImage: true,
	}

	if diff := cmp.Diff(got, want, cmp.AllowUnexported(buildermetrics.BuilderMetrics{}, buildermetrics.Counter{}, buildermetrics.FloatDP{}, buildererror.Error{}, buildermetadata.BuilderMetadata{})); diff != "" {
		t.Errorf("builder output parsing failed.  diff (-got +want):\n%v", diff)
	}
}

func TestJSON(t *testing.T) {
	bm := buildermetrics.NewBuilderMetrics()
	bm.GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(3)
	fm := buildermetadata.NewBuilderMetadata()
	fm.SetValue(buildermetadata.IsUsingGenkit, "true")
	fm.SetValue(buildermetadata.IsUsingGenAI, "false")
	fm.SetValue(buildermetadata.FrameworkName, "angular")
	fm.SetValue(buildermetadata.FrameworkVersion, "17.0.0")
	fm.SetValue(buildermetadata.AdapterName, "@apphosting/adapter-angular")
	fm.SetValue(buildermetadata.AdapterVersion, "17.2.3")
	fm.SetValue(buildermetadata.MonorepoName, "nx")
	fm.SetValue(buildermetadata.PackageManager, "pip")
	fm.SetValue(buildermetadata.ConfigFile, "requirements.txt")
	b := BuilderOutput{
		InstalledRuntimeVersions: []string{"6.0.6"},
		Metrics:                  bm,
		Error:                    buildererror.Error{Status: buildererror.StatusInternal},
		Metadata:                 fm,
	}

	s, err := b.JSON()

	if err != nil {
		t.Fatalf("Failed to marshal %v: %v", b, err)
	}
	if want := `"rtVersions":["6.0.6"]`; !strings.Contains(string(s), want) {
		t.Errorf("Expected string %q not found in %s", want, s)
	}
	if want := "INTERNAL"; !strings.Contains(string(s), want) {
		t.Errorf("Expected string %q not found in %s", want, s)
	}
	if want := `{"c":{"1":3}}`; !strings.Contains(string(s), want) {
		t.Errorf(`Expected string %q not found in %s`, want, s)
	}
	if want := `{"m":{"1":"true","2":"false","3":"angular","4":"17.0.0","5":"@apphosting/adapter-angular","6":"17.2.3","7":"nx","8":"pip","9":"requirements.txt"}}`; !strings.Contains(string(s), want) {
		t.Errorf(`Expected string %q not found in %s`, want, s)
	}
}

func TestFromJSONWithFetch(t *testing.T) {
	serialized := `
{
	"fetch": {
		"status": "SUCCESS",
		"sourceType": "ZipArchive",
		"location": "gs://test-bucket/src.zip#123",
		"totalDurationMs": 680,
		"downloadDurationMs": 460,
		"unzipDurationMs": 220,
		"downloadBytes": 1048576,
		"filesCount": 42
	},
	"rtVersions": ["24.0.0"],
	"stats": [
		{
			"buildpackId": "google.nodejs.npm",
			"buildpackVersion": "0.9.0",
			"totalDurationMs": 5000,
			"userDurationMs": 4500
		}
	]
}
`
	got, err := FromJSON([]byte(serialized))
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	want := BuilderOutput{
		Fetch: &FetchOutput{
			Status:             "SUCCESS",
			SourceType:         "ZipArchive",
			Location:           "gs://test-bucket/src.zip#123",
			TotalDurationMs:    680,
			DownloadDurationMs: 460,
			UnzipDurationMs:    220,
			DownloadBytes:      1048576,
			FilesCount:         42,
		},
		InstalledRuntimeVersions: []string{"24.0.0"},
		Stats: []BuilderStat{
			{
				BuildpackID:      "google.nodejs.npm",
				BuildpackVersion: "0.9.0",
				DurationMs:       5000,
				UserDurationMs:   4500,
			},
		},
	}

	if diff := cmp.Diff(got, want, cmp.AllowUnexported(buildermetrics.BuilderMetrics{}, buildermetrics.Counter{}, buildermetrics.FloatDP{}, buildererror.Error{}, buildermetadata.BuilderMetadata{})); diff != "" {
		t.Errorf("FromJSON with fetch diff (-got +want):\n%v", diff)
	}
}

func TestJSONWithFetch(t *testing.T) {
	b := BuilderOutput{
		Fetch: &FetchOutput{
			Status:          "SUCCESS",
			SourceType:      "ZipArchive",
			Location:        "gs://test-bucket/src.zip#123",
			TotalDurationMs: 680,
		},
	}

	s, err := b.JSON()
	if err != nil {
		t.Fatalf("Failed to marshal %v: %v", b, err)
	}
	if want := `"fetch":{"status":"SUCCESS","sourceType":"ZipArchive","location":"gs://test-bucket/src.zip#123","totalDurationMs":680,"downloadDurationMs":0,"unzipDurationMs":0,"downloadBytes":0,"filesCount":0}`; !strings.Contains(string(s), want) {
		t.Errorf("Expected fetch block not found in %s", s)
	}
}

func TestIsSystemError(t *testing.T) {
	testCases := []struct {
		name string
		bo   BuilderOutput
		want bool
	}{
		{
			name: "no match buildpack error",
			bo:   BuilderOutput{Error: buildererror.Error{Type: buildererror.StatusInvalidArgument}},
			want: false,
		},
		{
			name: "exact buildpack error",
			bo:   BuilderOutput{Error: buildererror.Error{Type: buildererror.StatusInternal}},
			want: true,
		},
		{
			name: "fetch user error",
			bo: BuilderOutput{
				Fetch: &FetchOutput{
					Error: &buildererror.Error{Type: buildererror.StatusPermissionDenied},
				},
			},
			want: false,
		},
		{
			name: "fetch internal system error",
			bo: BuilderOutput{
				Fetch: &FetchOutput{
					Error: &buildererror.Error{Type: buildererror.StatusInternal},
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := tc.bo.IsSystemError(), tc.want; got != want {
				t.Errorf("incorrect result for %q got=%t want=%t", tc.name, got, want)
			}
		})
	}
}
