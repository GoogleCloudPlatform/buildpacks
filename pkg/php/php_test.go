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

package php

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestReadComposerJSON(t *testing.T) {
	d, err := ioutil.TempDir("/tmp", "test-read-composer-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(d)

	contents := strings.TrimSpace(`
{
  "require": {
    "myorg/mypackage": "^0.7",
    "php": "7.4"
  },
  "scripts": {
    "gcp-build": "my-script"
  }
}
`)

	if err := ioutil.WriteFile(filepath.Join(d, composerJSON), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write composer.json: %v", err)
	}

	want := ComposerJSON{
		Require: map[string]string{
			"myorg/mypackage": "^0.7",
			"php":             "7.4",
		},
		Scripts: composerScriptsJSON{
			GCPBuild: "my-script",
		},
	}
	got, err := ReadComposerJSON(d)
	if err != nil {
		t.Errorf("ReadComposerJSON got error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadComposerJSON\ngot %#v\nwant %#v", *got, want)
	}
}

func TestReadComposerOverridesJSON(t *testing.T) {
	d, err := ioutil.TempDir("/tmp", "test-read-composer-overrides-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(d)

	contents := strings.TrimSpace(`
{
    "require": {
        "php": "^8.3",
        "myorg/mypackage": "^0.7"
    },
    "scripts": {
        "gcp-build": "my-script"
    },
    "extra": {
        "laravel": {
            "dont-discover": [
                "myorg/mypackage"
            ]
        },
        "google-buildpacks": {
            "document_root": "public",
            "front_controller": "index.php",
            "php-fpm": {
                "enable_dynamic_workers": true,
                "workers": 4
            },
            "serve_static": true
        }
    }
}
`)

	if err := ioutil.WriteFile(filepath.Join(d, composerJSON), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write composer.json: %v", err)
	}

	want := ComposerJSON{
		Require: map[string]string{
			"php":             "^8.3",
			"myorg/mypackage": "^0.7",
		},
		Scripts: composerScriptsJSON{
			GCPBuild: "my-script",
		},
		Extra: composerExtraJSON{
			GoogleBuildpacks: composerExtraGoogleBuildpacksJSON{
				DocumentRoot:    "public",
				FrontController: "index.php",
				PHPFPM: composerExtraGoogleBuildpacksPHPFPMJSON{
					EnableDynamicWorkers: true,
					Workers:              4,
				},
				ServeStatic: true,
			},
		},
	}
	got, err := ReadComposerJSON(d)
	if err != nil {
		t.Errorf("ReadComposerJSON got error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadComposerJSON\ngot %#v\nwant %#v", *got, want)
	}
}

func TestExtractVersion(t *testing.T) {

	testCases := []struct {
		name         string
		runtimeEnv   string
		want         string
		composerJSON string
		wantErr      bool
	}{
		{
			name:       "from environment",
			runtimeEnv: "7.3",
			want:       "7.3",
		},
		{
			name: "from composer.json",
			composerJSON: strings.TrimSpace(`
{
  "require": {
    "php": "7.4.1"
  }
}
`),
			want: "7.4.1",
		},
		{
			name:       "both environment and composer.json",
			runtimeEnv: "7.4.1",
			composerJSON: strings.TrimSpace(`
{
  "require": {
    "php": "7.3.0"
  }
}
`),
			want: "7.4.1",
		},
		{
			name: "composer.json without php version",
			composerJSON: strings.TrimSpace(`
{
  "require": {
    "myorg/mypackage": "^0.7"
  }
}
`),
			want: "",
		},
		{
			name: "invalid composer.json missing parentheses",
			composerJSON: strings.TrimSpace(`
{
  "require":
    "myorg/mypackage": "^0.7"
  }
}
`),
			wantErr: true,
		},
		{
			name: "no composer.json and environment",
			want: "",
		},
		{
			name: "composer.json with version constraint",
			composerJSON: strings.TrimSpace(`
{
  "require": {
    "php": ">=7.0.0"
  }
}
`),
			want: ">=7.0.0",
		},
		{
			name: "composer.json with complex version constraint",
			composerJSON: strings.TrimSpace(`
{
  "require": {
    "php": ">= 7.1.3, < 7.4.4"
  }
}
`),
			want: ">= 7.1.3, < 7.4.4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.runtimeEnv != "" {
				t.Setenv(env.RuntimeVersion, tc.runtimeEnv)
			}

			path, err := ioutil.TempDir("/tmp", "test-detect-version-")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(path)

			if len(tc.composerJSON) > 0 {
				if err := ioutil.WriteFile(filepath.Join(path, composerJSON), []byte(tc.composerJSON), 0644); err != nil {
					t.Fatalf("Failed to write composer.json: %v", err)
				}
			}

			ctx := gcp.NewContext(gcp.WithApplicationRoot(path))
			got, err := ExtractVersion(ctx)
			gotErr := err != nil

			if gotErr != tc.wantErr {
				t.Fatalf("ExtractVersion() got err=%t, want err=%t. err: %v", gotErr, tc.wantErr, err)
			}

			if got != tc.want {
				t.Errorf("ExtractVersion()=%q, want=%q", got, tc.want)
			}
		})
	}

}
