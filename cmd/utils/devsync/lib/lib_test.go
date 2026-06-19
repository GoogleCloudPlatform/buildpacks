// Copyright 2026 Google LLC
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

package lib

import (
	"os"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name   string
		envKey string
		envVal string
		want   bool
		opts   []gcp.ContextOption
	}{
		{
			name:   "with GOOGLE_DEVSYNC=true",
			envKey: "GOOGLE_DEVSYNC",
			envVal: "true",
			want:   true,
		},
		{
			name:   "without GOOGLE_DEVSYNC",
			envKey: "",
			envVal: "",
			want:   false,
		},
		{
			name:   "with GOOGLE_DEVSYNC=false",
			envKey: "GOOGLE_DEVSYNC",
			envVal: "false",
			want:   false,
		},
		{
			name:   "with Maker capability SkipDevsyncCapability",
			envKey: "GOOGLE_DEVSYNC",
			envVal: "true",
			want:   false,
			opts:   []gcp.ContextOption{gcp.WithCapability(SkipDevsyncCapability, true)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := append(tc.opts, gcp.WithApplicationRoot(testdata.MustGetPath("testdata/empty")))
			ctx := gcp.NewContext(opts...)
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			} else {
				os.Unsetenv("GOOGLE_DEVSYNC")
			}
			res, err := DetectFn(ctx)
			if err != nil {
				t.Fatalf("DetectFn failed: %v", err)
			}
			if res.Result().Pass != tc.want {
				t.Errorf("DetectFn Pass = %v, want %v", res.Result().Pass, tc.want)
			}
		})
	}
}
