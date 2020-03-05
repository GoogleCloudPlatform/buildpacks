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

package appengine

import (
	"os"
	"reflect"
	"testing"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpack/libbuildpack/buildpack"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name          string
		entrypointEnv string
		runtimeEnv    string
		mainEnv       string
		want          Config
	}{
		{
			name:          "entrypoint from env",
			entrypointEnv: "custom entrypoint",
			want: Config{
				Runtime: "runtime",
				Entrypoint: Entrypoint{
					Type:    EntrypointUser.String(),
					Command: "custom entrypoint",
				},
				MainExecutable: "",
			},
		},
		{
			name:       "runtime from env",
			runtimeEnv: "custom runtime",
			want: Config{
				Runtime: "custom runtime",
				Entrypoint: Entrypoint{
					Type:    EntrypointGenerated.String(),
					Command: "generated",
				},
				MainExecutable: "",
			},
		},

		{
			name:    "main from env",
			mainEnv: "custom main",
			want: Config{
				Runtime: "runtime",
				Entrypoint: Entrypoint{
					Type:    EntrypointGenerated.String(),
					Command: "generated",
				},
				MainExecutable: "custom main",
			},
		},
	}

	ctx := gcp.NewContext(buildpack.Info{ID: "id", Version: "version", Name: "name"})
	eg := func(*gcp.Context) (*Entrypoint, error) {
		return &Entrypoint{
			Type:    EntrypointGenerated.String(),
			Command: "generated",
		}, nil
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := setEnv("GOOGLE_ENTRYPOINT", tc.entrypointEnv)
			defer cleanup()
			cleanup = setEnv("GOOGLE_RUNTIME", tc.runtimeEnv)
			defer cleanup()
			cleanup = setEnv("GAE_YAML_MAIN", tc.mainEnv)
			defer cleanup()

			got, err := getConfig(ctx, "runtime", eg)
			if err != nil {
				t.Errorf("getConfig() got error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("getConfig() got=%#v, want=%#v", got, tc.want)
			}
		})
	}
}

func setEnv(key, value string) func() {
	var cleanup func()
	if val, ok := os.LookupEnv(key); ok {
		cleanup = func() {
			os.Setenv(key, val)
		}
	} else {
		cleanup = func() {
			os.Unsetenv(key)
		}
	}
	os.Setenv(key, value)
	return cleanup
}
