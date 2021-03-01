// Copyright 2021 Google LLC
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

package cloudfunctions

import (
	"os"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appstart"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

func TestGetConfig(t *testing.T) {
	testCases := []struct {
		name       string
		runtimeEnv string
		want       appstart.Config
	}{
		{
			name: "default",
			want: appstart.Config{
				Runtime: "runtime",
				Entrypoint: appstart.Entrypoint{
					Type:    appstart.EntrypointGenerated.String(),
					Command: "generated",
				},
			},
		},
		{
			name:       "runtime from env",
			runtimeEnv: "custom runtime",
			want: appstart.Config{
				Runtime: "custom runtime",
				Entrypoint: appstart.Entrypoint{
					Type:    appstart.EntrypointGenerated.String(),
					Command: "generated",
				},
			},
		},
	}

	ctx := gcp.NewContext(libcnb.BuildpackInfo{ID: "id", Version: "version", Name: "name"})
	eg := func(*gcp.Context) (*appstart.Entrypoint, error) {
		return &appstart.Entrypoint{
			Type:    appstart.EntrypointGenerated.String(),
			Command: "generated",
		}, nil
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, "GOOGLE_RUNTIME", tc.runtimeEnv)

			got, err := getConfig(ctx, "runtime", eg)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func setEnv(t *testing.T, name, value string) {
	t.Helper()

	old, oldPresent := os.LookupEnv(name)
	if err := os.Setenv(name, value); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if oldPresent {
			if err := os.Setenv(name, old); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	})
}
