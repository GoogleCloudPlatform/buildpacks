// Copyright 2022 Google LLC
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

package dart

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/testserver"
)

func TestResolveFlutterPackageVersion(t *testing.T) {
	testCases := []struct {
		name       string
		env        string
		httpStatus int
		response   string
		want       string
		archive    string
		wantError  bool
	}{
		{
			name: "from env",
			env:  "3.32.0-0.5.pre",
			response: `{
  "base_url": "https://storage.googleapis.com/flutter_infra_release/releases",
  "current_release": {
    "beta": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
    "dev": "13a2fb10b838971ce211230f8ffdd094c14af02c",
    "stable": "ea121f8859e4b13e47a8f845e4586164519588bc"
  },
  "releases": [
    {
      "hash": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
      "channel": "beta",
      "version": "3.32.0-0.5.pre",
      "dart_sdk_version": "3.8.0",
      "dart_sdk_arch": "x64",
      "release_date": "2025-05-16T19:00:28.219866Z",
      "archive": "beta/linux/flutter_linux_3.32.0-0.5.pre-beta.tar.xz",
      "sha256": "c7833044a7954aed020b54057fd80eeb39c8655c505d7e5896c9c13a7c10713b"
    },
    {
      "hash": "ea121f8859e4b13e47a8f845e4586164519588bc",
      "channel": "stable",
      "version": "3.29.3",
      "dart_sdk_version": "3.7.2",
      "dart_sdk_arch": "x64",
      "release_date": "2025-04-14T17:25:51.061305Z",
      "archive": "stable/linux/flutter_linux_3.29.3-stable.tar.xz",
      "sha256": "8a908a5add53c1dfc2031da29e58daefd59a6d1d52fb5cb61f5ee52c73e36e15"
    },
    {
      "hash": "c23637390482d4cf9598c3ce3f2be31aa7332daf",
      "channel": "stable",
      "version": "3.29.2",
      "dart_sdk_version": "3.7.2",
      "dart_sdk_arch": "x64",
      "release_date": "2025-03-14T14:20:23.400177Z",
      "archive": "stable/linux/flutter_linux_3.29.2-stable.tar.xz",
      "sha256": "6096f21370773093ec19240e133664c1c12eb8b5a85605a92d16ce462a18eac4"
    }	
  ]
}`,
			want:    "3.32.0-0.5.pre",
			archive: "beta/linux/flutter_linux_3.32.0-0.5.pre-beta.tar.xz",
		},
		{
			name: "fetched version",
			response: `{
  "base_url": "https://storage.googleapis.com/flutter_infra_release/releases",
  "current_release": {
    "beta": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
    "dev": "13a2fb10b838971ce211230f8ffdd094c14af02c",
    "stable": "ea121f8859e4b13e47a8f845e4586164519588bc"
  },
  "releases": [
    {
      "hash": "48ea72a87d7fc69d73aa2531ded8a5da9d13b2bd",
      "channel": "beta",
      "version": "3.32.0-0.5.pre",
      "dart_sdk_version": "3.8.0",
      "dart_sdk_arch": "x64",
      "release_date": "2025-05-16T19:00:28.219866Z",
      "archive": "beta/linux/flutter_linux_3.32.0-0.5.pre-beta.tar.xz",
      "sha256": "c7833044a7954aed020b54057fd80eeb39c8655c505d7e5896c9c13a7c10713b"
    },
    {
      "hash": "ea121f8859e4b13e47a8f845e4586164519588bc",
      "channel": "stable",
      "version": "3.29.3",
      "dart_sdk_version": "3.7.2",
      "dart_sdk_arch": "x64",
      "release_date": "2025-04-14T17:25:51.061305Z",
      "archive": "stable/linux/flutter_linux_3.29.3-stable.tar.xz",
      "sha256": "8a908a5add53c1dfc2031da29e58daefd59a6d1d52fb5cb61f5ee52c73e36e15"
    },
    {
      "hash": "c23637390482d4cf9598c3ce3f2be31aa7332daf",
      "channel": "stable",
      "version": "3.29.2",
      "dart_sdk_version": "3.7.2",
      "dart_sdk_arch": "x64",
      "release_date": "2025-03-14T14:20:23.400177Z",
      "archive": "stable/linux/flutter_linux_3.29.2-stable.tar.xz",
      "sha256": "6096f21370773093ec19240e133664c1c12eb8b5a85605a92d16ce462a18eac4"
    }	
  ]
}`,
			want:    "3.29.3",
			archive: "stable/linux/flutter_linux_3.29.3-stable.tar.xz",
		},
		{
			name:       "bad response code",
			httpStatus: http.StatusBadRequest,
			wantError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			testserver.New(
				t,
				testserver.WithStatus(tc.httpStatus),
				testserver.WithJSON(tc.response),
				testserver.WithMockURL(&flutterVersionURL),
			)

			if tc.env != "" {
				t.Setenv("GOOGLE_RUNTIME_VERSION", tc.env)
			}

			got, archive, err := DetectFlutterSDKArchive()
			if tc.wantError == (err == nil) {
				t.Errorf(`DetectSDKArchive() got error: %v, want error?: %v`, err, tc.wantError)
			}
			if got != tc.want {
				t.Errorf(`DetectSDKArchive() = %q, want version %q`, got, tc.want)
			}
			if archive != tc.archive {
				t.Errorf(`DetectSDKArchive() = %q, want archive %q`, got, tc.archive)
			}
		})
	}
}

func TestIsFlutter(t *testing.T) {
	testCases := []struct {
		name    string
		pubspec string
		want    bool
		wantErr bool
	}{
		{
			name: "no pubspec.yaml",
		},
		{
			name:    "no dependencies",
			pubspec: `name: test`,
		},
		{
			name: "no flutter",
			pubspec: `
name: example_json_function

dependencies:
  functions_framework: ^0.4.0
`,
		},
		{
			name: "with dev_dependency",
			pubspec: `
name: example_json_function

dependencies:
  flutter:
    sdk: flutter
`,
			want: true,
		},
		{
			name:    "invalid yaml",
			pubspec: "\t",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.pubspec != "" {
				path := filepath.Join(dir, "pubspec.yaml")
				if err := os.WriteFile(path, []byte(tc.pubspec), 0744); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}
			got, err := IsFlutter(dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("IsFlutter(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("IsFlutter(%q) = %t, want %t", dir, got, tc.want)
			}
		})
	}
}

func TestGetPubspec(t *testing.T) {
	testCases := []struct {
		name    string
		pubspec string
		want    *Buildpack
		wantErr bool
	}{
		{
			name: "no pubspec.yaml",
		},
		{
			name:    "no dependencies",
			pubspec: `name: test`,
		},
		{
			name: "no buildpack key",
			pubspec: `
name: example_app

dependencies:
  functions_framework: ^0.4.0
`,
		},
		{
			name: "with buildpack defaults",
			pubspec: `
name: example_json_function

dependencies:
  flutter:
    sdk: flutter
buildpack:
  invalid_key: test
`,
			want: &Buildpack{
				Server: ptr("server"),
				Static: ptr("static"),
			},
		},
		{
			name: "with buildpack specifics",
			pubspec: `
name: example_json_function

dependencies:
  flutter:
    sdk: flutter
buildpack:
  server: server_test
  static: static_test
`,
			want: &Buildpack{
				Server: ptr("server_test"),
				Static: ptr("static_test"),
			},
		},
		{
			name:    "invalid yaml",
			pubspec: "\t",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.pubspec != "" {
				path := filepath.Join(dir, "pubspec.yaml")
				if err := os.WriteFile(path, []byte(tc.pubspec), 0744); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}
			got, err := GetPubspec(dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("GetPubspec(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if (tc.want == nil && got.Buildpack != nil) || (tc.want != nil && got.Buildpack == nil) {
				t.Errorf("GetPubspec(%q) = want/got nil missmatch", dir)
			}
			if got.Buildpack != nil && *got.Buildpack.Server != *tc.want.Server {
				t.Errorf("GetPubspec(%q) = %q, want server %q", dir, *got.Buildpack.Server, *tc.want.Server)
			}
			if got.Buildpack != nil && *got.Buildpack.Static != *tc.want.Static {
				t.Errorf("GetPubspec(%q) = %q, want static %q", dir, *got.Buildpack.Static, *tc.want.Static)
			}
		})
	}
}

func ptr(s string) *string { return &s }
