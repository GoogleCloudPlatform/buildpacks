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
	"metadata": {"m":{"1":"true", "2":"false", "3":"angular", "4":"17.0.0", "5":"@apphosting/adapter-angular", "6":"17.2.3", "7":"nx"}},
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
	if want := `{"m":{"1":"true","2":"false","3":"angular","4":"17.0.0","5":"@apphosting/adapter-angular","6":"17.2.3","7":"nx"}}`; !strings.Contains(string(s), want) {
		t.Errorf(`Expected string %q not found in %s`, want, s)
	}
}

func TestIsSystemError(t *testing.T) {
	testCases := []struct {
		name      string
		errorType buildererror.Status
		want      bool
	}{
		{
			name:      "no match",
			errorType: buildererror.StatusInvalidArgument,
			want:      false,
		},
		{
			name:      "exact",
			errorType: buildererror.StatusInternal,
			want:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bo := BuilderOutput{Error: buildererror.Error{Type: tc.errorType}}

			if got, want := bo.IsSystemError(), tc.want; got != want {
				t.Errorf("incorrect result for %q got=%t want=%t", tc.errorType, got, want)
			}
		})
	}
}
