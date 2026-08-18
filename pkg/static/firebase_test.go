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

package static

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

func TestParseFirebaseConfig(t *testing.T) {
	ptrBool := func(b bool) *bool { return &b }

	tests := []struct {
		name        string
		fileContent string // Non-empty will create a temp file; empty represents a missing file
		wantConfigs []HostingConfig
		wantErr     bool
	}{
		{
			name:        "not found",
			fileContent: "",
			wantConfigs: nil,
		},
		{
			name: "single hosting object",
			fileContent: `{
				"hosting": {
					"public": "dist",
					"cleanUrls": true,
					"trailingSlash": false,
					"rewrites": [{"source": "**", "destination": "/index.html"}],
					"redirects": [{"source": "/old", "destination": "/new", "type": 301}],
					"headers": [{"source": "**/*.css", "headers": [{"key": "Cache-Control", "value": "max-age=31536000"}]}]
				}
			}`,
			wantConfigs: []HostingConfig{
				{
					Public:        "dist",
					CleanUrls:     true,
					TrailingSlash: ptrBool(false),
					Rewrites: []Rewrite{
						{Source: "**", Destination: "/index.html"},
					},
					Redirects: []Redirect{
						{Source: "/old", Destination: "/new", Type: 301},
					},
					Headers: []Header{
						{
							Source: "**/*.css",
							Headers: []HeaderConfig{
								{Key: "Cache-Control", Value: "max-age=31536000"},
							},
						},
					},
				},
			},
		},
		{
			name: "array hosting objects",
			fileContent: `{
				"hosting": [
					{"target": "app1", "public": "dist1"},
					{"target": "app2", "public": "dist2"}
				]
			}`,
			wantConfigs: []HostingConfig{
				{Target: "app1", Public: "dist1"},
				{Target: "app2", Public: "dist2"},
			},
		},
		{
			name: "hosting with function object and run tag",
			fileContent: `{
				"hosting": {
					"public": "public",
					"rewrites": [
						{
							"source": "/compute",
							"function": {
								"functionId": "heavyCompute",
								"region": "europe-west1"
							}
						},
						{
							"source": "/api",
							"run": {
								"serviceId": "user-service",
								"region": "us-central1",
								"tag": "canary"
							}
						}
					]
				}
			}`,
			wantConfigs: []HostingConfig{
				{
					Public: "public",
					Rewrites: []Rewrite{
						{
							Source:         "/compute",
							Function:       "heavyCompute",
							FunctionRegion: "europe-west1",
						},
						{
							Source: "/api",
							Run: &Run{
								ServiceID: "user-service",
								Region:    "us-central1",
								Tag:       "canary",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := "nonexistent.json"
			if tc.fileContent != "" {
				tmpDir := t.TempDir()
				p = filepath.Join(tmpDir, "firebase.json")
				if err := os.WriteFile(p, []byte(tc.fileContent), 0644); err != nil {
					t.Fatalf("writing temp file: %v", err)
				}
			}

			got, err := ParseFirebaseConfig(p)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseFirebaseConfig(%q) returned error %v, wantErr %t", p, err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantConfigs, got); diff != "" {
				t.Errorf("ParseFirebaseConfig(%q) returned unexpected diff (-want +got):\n%s", p, diff)
			}
		})
	}
}

func TestGlobToRegex(t *testing.T) {
	testCases := []struct {
		name           string
		glob           string
		want           string
		shouldMatch    []string
		shouldNotMatch []string
		wantErr        bool
	}{
		{
			name:        "double_star",
			glob:        "**",
			want:        "^.*$",
			shouldMatch: []string{"/", "/foo", "/foo/bar", "/foo/bar.js"},
		},
		{
			name:           "double_star_extension",
			glob:           "**/*.js",
			want:           "^.*/[^/]*\\.js$",
			shouldMatch:    []string{"/foo.js", "/foo/bar.js", "/foo/bar/baz.js"},
			shouldNotMatch: []string{"/foo.js.map", "/foo.jsx", "/foo/bar"},
		},
		{
			name:           "single_star_extension",
			glob:           "*.html",
			want:           "^/[^/]*\\.html$",
			shouldMatch:    []string{"/index.html", "/about.html"},
			shouldNotMatch: []string{"/foo/index.html", "/index.htm"},
		},
		{
			name:           "extglob_at",
			glob:           "tags/@(a|b)",
			want:           "^/tags/(?:a|b)$",
			shouldMatch:    []string{"/tags/a", "/tags/b"},
			shouldNotMatch: []string{"/tags/c", "/tags/a/b"},
		},
		{
			name:           "exact_path",
			glob:           "exact-path",
			want:           "^/exact-path$",
			shouldMatch:    []string{"/exact-path"},
			shouldNotMatch: []string{"/exact-path/extra", "/other"},
		},
		{
			name:           "extglob_at_multi_extension",
			glob:           "**/*.@(jpg|jpeg|gif|png)",
			want:           `^.*/[^/]*\.(?:jpg|jpeg|gif|png)$`,
			shouldMatch:    []string{"/img.png", "/assets/img.jpeg", "/a/b/c/d.gif"},
			shouldNotMatch: []string{"/img.png.txt", "/img.webp"},
		},
		{
			name:           "extglob_nested_double_star",
			glob:           "tags/@(v1|v2)/**/*.@(js|css)",
			want:           `^/tags/(?:v1|v2)/(?:.*/)?[^/]*\.(?:js|css)$`,
			shouldMatch:    []string{"/tags/v1/a/b/c.js", "/tags/v2/styles.css"}, // tests 0-nested directory matching!
			shouldNotMatch: []string{"/tags/v3/a.js", "/tags/v1/a.html"},
		},
		{
			name:           "double_star_dir_single_star",
			glob:           "**/foo/*",
			want:           `^.*/foo/[^/]*$`,
			shouldMatch:    []string{"/foo/bar", "/a/b/foo/baz"},
			shouldNotMatch: []string{"/foo/bar/baz", "/a/foo"},
		},
		{
			name:           "question_mark_char",
			glob:           "file?.html",
			want:           `^/file[^/]\.html$`,
			shouldMatch:    []string{"/file1.html", "/fileA.html", "/file-.html"},
			shouldNotMatch: []string{"/file.html", "/file12.html", "/file/.html"},
		},
		{
			name:           "double_star_question_dir",
			glob:           "**/dir?/file.js",
			want:           `^.*/dir[^/]/file\.js$`,
			shouldMatch:    []string{"/dir1/file.js", "/a/b/dirA/file.js"},
			shouldNotMatch: []string{"/dir/file.js", "/dir12/file.js", "/a/b/dir/c/file.js"},
		},
		{
			name:           "bracket_digit_range",
			glob:           `**/chunk-[0-9].js`,
			want:           `^.*/chunk-[0-9]\.js$`,
			shouldMatch:    []string{"/chunk-5.js", "/a/b/chunk-0.js"},
			shouldNotMatch: []string{"/chunk-a.js", "/chunk-56.js", "/a/b/chunk-/.js"},
		},
		{
			name:           "bracket_negated_range",
			glob:           `**/*[!a-z].txt`,
			want:           `^.*/[^/]*[^/a-z]\.txt$`,
			shouldMatch:    []string{"/file1.txt", "/dir/file1.txt", "/dir/sub/file-1.txt", "/dir/file/1.txt"},
			shouldNotMatch: []string{"/filea.txt", "/dir/filea.txt", "/dir/file/.txt", "/dir/file1/a.txt"},
		},
		{
			name:           "bracket_escaped_hyphen",
			glob:           `file[a\-z].txt`,
			want:           `^/file[a\-z]\.txt$`,
			shouldMatch:    []string{"/filea.txt", "/file-.txt", "/filez.txt"},
			shouldNotMatch: []string{"/fileb.txt", `/file\.txt`},
		},
		{
			name:           "bracket_escaped_closing",
			glob:           `file[a-z\].txt`, // unbalanced because \ escapes ]
			want:           `^/file\[a-z\]\.txt$`,
			shouldMatch:    []string{"/file[a-z].txt"},
			shouldNotMatch: []string{"/file[a-z\\].txt", "/filea.txt"},
		},
		{
			name:           "bracket_escaped_backslash",
			glob:           `file[a-z\\].txt`, // balanced, matches a-z or \
			want:           `^/file[a-z\\]\.txt$`,
			shouldMatch:    []string{"/filea.txt", "/file\\.txt"},
			shouldNotMatch: []string{"/file[a-z].txt", "/file1.txt"},
		},
		{
			name:           "bracket_complex_escaped",
			glob:           `file[a-z\\\]].txt`, // balanced, matches a-z, \ or ]
			want:           `^/file[a-z\\\]]\.txt$`,
			shouldMatch:    []string{"/filea.txt", "/file\\.txt", "/file].txt"},
			shouldNotMatch: []string{"/file[a-z].txt", "/file1.txt"},
		},
		{
			name:        "bracket_empty",
			glob:        `file[].txt`, // empty class, treated as literal
			want:        `^/file\[\]\.txt$`,
			shouldMatch: []string{"/file[].txt"},
		},
		{
			name:        "bracket_unbalanced",
			glob:        `file[.txt`, // unbalanced, treated as literal
			want:        `^/file\[\.txt$`,
			shouldMatch: []string{"/file[.txt"},
		},
		{
			name:           "extglob_star",
			glob:           `tags/*(a|b)`,
			want:           `^/tags/(?:a|b)*$`,
			shouldMatch:    []string{"/tags/", "/tags/a", "/tags/b", "/tags/ab", "/tags/aaaa", "/tags/abab"},
			shouldNotMatch: []string{"/tags/c", "/tags/a/b"},
		},
		{
			name:           "extglob_plus",
			glob:           `tags/+(a|b)`,
			want:           `^/tags/(?:a|b)+$`,
			shouldMatch:    []string{"/tags/a", "/tags/b", "/tags/ab", "/tags/aaaa", "/tags/abab"},
			shouldNotMatch: []string{"/tags/", "/tags/c", "/tags/a/b"},
		},
		{
			name:           "extglob_question",
			glob:           `tags/?(a|b)`,
			want:           `^/tags/(?:a|b)?$`,
			shouldMatch:    []string{"/tags/", "/tags/a", "/tags/b"},
			shouldNotMatch: []string{"/tags/ab", "/tags/c", "/tags/a/b"},
		},
		{
			name:           "extglob_nested",
			glob:           `tags/*(a|+(b|c))`,
			want:           `^/tags/(?:a|(?:b|c)+)*$`,
			shouldMatch:    []string{"/tags/", "/tags/a", "/tags/b", "/tags/c", "/tags/bc", "/tags/abbc", "/tags/bbccaa"},
			shouldNotMatch: []string{"/tags/d", "/tags/a/b"},
		},
		{
			name:           "extglob_exclamation",
			glob:           `tags/!(a|b)`,
			want:           `^/tags/(?!(?:a|b)(?:/|$))[^/]*$`,
			shouldMatch:    []string{"/tags/", "/tags/c", "/tags/ab", "/tags/aaaa"},
			shouldNotMatch: []string{"/tags/a", "/tags/b"},
		},
		{
			name:           "extglob_exclamation_subpath",
			glob:           `tags/!(a|b)/src`,
			want:           `^/tags/(?!(?:a|b)(?:/|$))[^/]*/src$`,
			shouldMatch:    []string{"/tags/c/src", "/tags/ab/src"},
			shouldNotMatch: []string{"/tags/a/src", "/tags/b/src"},
		},
		{
			name:    "unbalanced_extglob_prefix",
			glob:    "/unbalanced/@(a|b",
			wantErr: true,
		},
		{
			name:    "unbalanced_extglob_mid",
			glob:    `tags/unbalanced/*(a|b`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GlobToRegex(tc.glob)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GlobToRegex(%q) error = %v, wantErr %v", tc.glob, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got != tc.want {
				t.Errorf("GlobToRegex(%q) = %q, want %q", tc.glob, got, tc.want)
			}

			// Go's RE2 engine does not support PCRE lookaheads (?!).
			// We skip running the regex in Go if it contains lookaheads,
			// but we still verified that the generated string (got) matches the expected (tc.want).
			if strings.Contains(got, "(?!") {
				return
			}

			re, err := regexp.Compile("(?i)" + got) // Nginx matches case-insensitively
			if err != nil {
				t.Fatalf("Compiled regex %q is invalid: %v", got, err)
			}

			for _, m := range tc.shouldMatch {
				if !re.MatchString(m) {
					t.Errorf("Regex %q should match %q", got, m)
				}
			}
			for _, f := range tc.shouldNotMatch {
				if re.MatchString(f) {
					t.Errorf("Regex %q should NOT match %q", got, f)
				}
			}
		})
	}
}

func TestPrepareNginxHeaders_CustomHeaders(t *testing.T) {
	hostingConfig := &HostingConfig{
		Headers: []Header{
			{
				Source:  "/exact",
				Headers: []HeaderConfig{{Key: "Cache-Control", Value: "no-store"}},
			},
			{
				Source: "**/*.png",
				Headers: []HeaderConfig{
					{Key: "cache-control", Value: "max-age=100"},
					{Key: "X-Foo", Value: `bar "with" \escapes`},
				},
			},
		},
	}

	gotMaps, gotHeaders, err := PrepareNginxHeaders(hostingConfig)
	if err != nil {
		t.Fatalf("PrepareNginxHeaders() error = %v", err)
	}

	wantMaps := []FirebaseNginxMap{
		{
			VariableName: "hdr_cache_control",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^.*/[^/]*\.png$`, Value: `"max-age=100"`},
				{Regex: `^/exact$`, Value: `"no-store"`},
			},
		},
		{
			VariableName: "hdr_x_foo",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^.*/[^/]*\.png$`, Value: `"bar \"with\" \\escapes"`},
			},
		},
	}
	wantHeaders := []FirebaseNginxHeader{
		{Key: "Cache-Control", VariableName: "hdr_cache_control"},
		{Key: "X-Foo", VariableName: "hdr_x_foo"},
	}

	if diff := cmp.Diff(wantMaps, gotMaps); diff != "" {
		t.Errorf("PrepareNginxHeaders() maps mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantHeaders, gotHeaders); diff != "" {
		t.Errorf("PrepareNginxHeaders() headers mismatch (-want +got):\n%s", diff)
	}
}

func TestPrepareNginxHeaders_MultiValueAndEscaping(t *testing.T) {
	hostingConfig := &HostingConfig{
		Headers: []Header{
			{
				Source: "/session",
				Headers: []HeaderConfig{
					{Key: "Set-Cookie", Value: "session_id=123; Secure"},
					{Key: "Set-Cookie", Value: "user=john; Path=/"},
					{Key: "X-Custom", Value: "value$with$dollars"},
				},
			},
			{
				Source: "/tracking",
				Headers: []HeaderConfig{
					{Key: "Set-Cookie", Value: "track=abc"},
				},
			},
		},
	}

	gotMaps, gotHeaders, err := PrepareNginxHeaders(hostingConfig)
	if err != nil {
		t.Fatalf("PrepareNginxHeaders() error = %v", err)
	}

	wantMaps := []FirebaseNginxMap{
		{
			VariableName: "hdr_x_custom",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/session$`, Value: `"value${literal_dollar}with${literal_dollar}dollars"`},
			},
		},
		{
			VariableName: "hdr_set_cookie_0",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/tracking$`, Value: `"track=abc"`},
			},
		},
		{
			VariableName: "hdr_set_cookie_1",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/session$`, Value: `"user=john; Path=/"`},
			},
		},
		{
			VariableName: "hdr_set_cookie_2",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/session$`, Value: `"session_id=123; Secure"`},
			},
		},
	}
	wantHeaders := []FirebaseNginxHeader{
		{Key: "X-Custom", VariableName: "hdr_x_custom"},
		{Key: "Set-Cookie", VariableName: "hdr_set_cookie_0"},
		{Key: "Set-Cookie", VariableName: "hdr_set_cookie_1"},
		{Key: "Set-Cookie", VariableName: "hdr_set_cookie_2"},
	}

	if diff := cmp.Diff(wantMaps, gotMaps); diff != "" {
		t.Errorf("PrepareNginxHeaders() maps mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantHeaders, gotHeaders); diff != "" {
		t.Errorf("PrepareNginxHeaders() headers mismatch (-want +got):\n%s", diff)
	}
}

func TestPrepareNginxHeaders_ComprehensiveHeaders(t *testing.T) {
	hostingConfig := &HostingConfig{
		Headers: []Header{
			{
				Source:  "**/*.@(js|css)",
				Headers: []HeaderConfig{{Key: "Cache-Control", Value: "max-age=31536000"}},
			},
			{
				Source:  "**/chunk-[0-9].js",
				Headers: []HeaderConfig{{Key: "X-Chunk", Value: "true"}},
			},
			{
				Source:  "tags/!(a|b)/*.png",
				Headers: []HeaderConfig{{Key: "X-Tag", Value: "safe"}},
			},
		},
	}

	gotMaps, gotHeaders, err := PrepareNginxHeaders(hostingConfig)
	if err != nil {
		t.Fatalf("PrepareNginxHeaders() error = %v", err)
	}

	wantMaps := []FirebaseNginxMap{
		{
			VariableName: "hdr_cache_control",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^.*/[^/]*\.(?:js|css)$`, Value: `"max-age=31536000"`},
			},
		},
		{
			VariableName: "hdr_x_chunk",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^.*/chunk-[0-9]\.js$`, Value: `"true"`},
			},
		},
		{
			VariableName: "hdr_x_tag",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/tags/(?!(?:a|b)(?:/|$))[^/]*/[^/]*\.png$`, Value: `"safe"`},
			},
		},
	}
	wantHeaders := []FirebaseNginxHeader{
		{Key: "Cache-Control", VariableName: "hdr_cache_control"},
		{Key: "X-Chunk", VariableName: "hdr_x_chunk"},
		{Key: "X-Tag", VariableName: "hdr_x_tag"},
	}

	if diff := cmp.Diff(wantMaps, gotMaps); diff != "" {
		t.Errorf("PrepareNginxHeaders() maps mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantHeaders, gotHeaders); diff != "" {
		t.Errorf("PrepareNginxHeaders() headers mismatch (-want +got):\n%s", diff)
	}
}

func TestPrepareNginxHeaders_DuplicateHeaders(t *testing.T) {
	hostingConfig := &HostingConfig{
		Headers: []Header{
			{
				Source: "/exact",
				Headers: []HeaderConfig{
					{Key: "X-First", Value: "value1"},
					{Key: "X-First", Value: "value2"}, // duplicate in same block, should win
					{Key: "X-Second", Value: "block1-val"},
				},
			},
			{
				Source: "/exact",
				Headers: []HeaderConfig{
					{Key: "X-Second", Value: "block2-val"}, // duplicate across blocks, should win
				},
			},
		},
	}

	gotMaps, gotHeaders, err := PrepareNginxHeaders(hostingConfig)
	if err != nil {
		t.Fatalf("PrepareNginxHeaders() error = %v", err)
	}

	wantMaps := []FirebaseNginxMap{
		{
			VariableName: "hdr_x_first",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/exact$`, Value: `"value2"`},
			},
		},
		{
			VariableName: "hdr_x_second",
			DefaultValue: `""`,
			Mappings: []FirebaseNginxMapping{
				{Regex: `^/exact$`, Value: `"block2-val"`},
			},
		},
	}
	wantHeaders := []FirebaseNginxHeader{
		{Key: "X-First", VariableName: "hdr_x_first"},
		{Key: "X-Second", VariableName: "hdr_x_second"},
	}

	if diff := cmp.Diff(wantMaps, gotMaps); diff != "" {
		t.Errorf("PrepareNginxHeaders() maps mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantHeaders, gotHeaders); diff != "" {
		t.Errorf("PrepareNginxHeaders() headers mismatch (-want +got):\n%s", diff)
	}
}

func TestWriteFirebaseNginxConfig_GoldenFile(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, NginxConfFile)

	hostingConfig := &HostingConfig{
		Headers: []Header{
			{
				Source:  "/exact",
				Headers: []HeaderConfig{{Key: "Cache-Control", Value: "no-store"}},
			},
			{
				Source: "**/*.png",
				Headers: []HeaderConfig{
					{Key: "cache-control", Value: "max-age=100"},
					{Key: "X-Foo", Value: `bar "with" \escapes`},
				},
			},
			{
				Source: "/session",
				Headers: []HeaderConfig{
					{Key: "Set-Cookie", Value: "session_id=123; Secure"},
					{Key: "Set-Cookie", Value: "user=john; Path=/"},
					{Key: "X-Custom", Value: "value$with$dollars"},
				},
			},
		},
	}
	maps, headers, err := PrepareNginxHeaders(hostingConfig)
	if err != nil {
		t.Fatalf("PrepareNginxHeaders() error = %v", err)
	}

	params := FirebaseNginxConfigParams{
		RootPath:      "/my/app/root",
		MimeTypesPath: "/opt/nginx/conf/mime.types",
		Maps:          maps,
		Headers:       headers,
	}

	if err := WriteFirebaseNginxConfig(dstPath, params); err != nil {
		t.Fatalf("WriteFirebaseNginxConfig() error = %v", err)
	}

	gotBytes, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", dstPath, err)
	}

	wantBytes, err := os.ReadFile(testdata.MustGetPath("testdata/firebase_nginx_expected.conf"))
	if err != nil {
		t.Fatalf("os.ReadFile(firebase_nginx_expected.conf) error = %v", err)
	}

	if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
		t.Errorf("WriteFirebaseNginxConfig() generated config mismatch (-want +got):\n%s", diff)
	}
}

func TestWriteFirebaseNginxConfig_RedirectsAndRewrites(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, NginxConfFile)

	params := FirebaseNginxConfigParams{
		RootPath:      "/my/app/root",
		MimeTypesPath: "/opt/nginx/conf/mime.types",
		Redirects: []NginxRedirect{
			{Pattern: `^/old$`, Target: "/new", Code: 301},
		},
		Rewrites: []NginxRewrite{
			{Pattern: `^/api/(.*)$`, Target: "/index.html"},
		},
	}

	if err := WriteFirebaseNginxConfig(dstPath, params); err != nil {
		t.Fatalf("WriteFirebaseNginxConfig() error = %v", err)
	}

	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", dstPath, err)
	}

	got := string(content)
	if !strings.Contains(got, "location ~ ^/old$ {") {
		t.Errorf("WriteFirebaseNginxConfig() output = %q; missing redirect location block", got)
	}
	if !strings.Contains(got, "return 301 /new;") {
		t.Errorf("WriteFirebaseNginxConfig() output = %q; missing redirect return directive", got)
	}
	if !strings.Contains(got, `location ~ ^/api/(.*)$ {`) {
		t.Errorf("WriteFirebaseNginxConfig() output = %q; missing rewrite location block", got)
	}
	if !strings.Contains(got, `rewrite ^/api/(.*)$ /index.html break;`) {
		t.Errorf("WriteFirebaseNginxConfig() output = %q; missing rewrite directive", got)
	}
}

func TestTranslateRedirects(t *testing.T) {
	tests := []struct {
		name        string
		fbRedirects []Redirect
		want        []NginxRedirect
		wantErr     bool
	}{
		{
			name: "basic redirect",
			fbRedirects: []Redirect{
				{Source: "/foo", Destination: "/bar", Type: 302},
			},
			want: []NginxRedirect{
				{Pattern: `^/foo$`, Target: "/bar", Code: 302},
			},
		},
		{
			name: "glob redirect",
			fbRedirects: []Redirect{
				{Source: "/blog/**", Destination: "/news", Type: 301},
			},
			want: []NginxRedirect{
				{Pattern: `^/blog(?:/.*)?$`, Target: "/news", Code: 301},
			},
		},
		{
			name: "segment redirect (splat)",
			fbRedirects: []Redirect{
				{Source: "/library-seg/:splat*", Destination: "/archive/:splat*", Type: 301},
			},
			want: []NginxRedirect{
				{Pattern: `^/library-seg/(?P<splat>.+)$`, Target: "/archive/$splat", Code: 301},
			},
		},
		{
			name: "regex redirect",
			fbRedirects: []Redirect{
				{Regex: `^/p/(.*)$`, Destination: "/post/$1"},
			},
			want: []NginxRedirect{
				{Pattern: `^/p/(.*)$`, Target: "/post/$1", Code: 301},
			},
		},
		{
			name: "redirect with colon segments",
			fbRedirects: []Redirect{
				{Source: "/blog/:slug", Destination: "/blogs/:slug", Type: 301},
			},
			want: []NginxRedirect{
				{Pattern: `^/blog/(?P<slug>[^/]+)$`, Target: "/blogs/$slug", Code: 301},
			},
		},
		{
			name: "redirect with regex and numeric captures",
			fbRedirects: []Redirect{
				{Regex: `^/p/(\d+)$`, Destination: "/post/:1"},
			},
			want: []NginxRedirect{
				{Pattern: `^/p/(\d+)$`, Target: "/post/$1", Code: 301},
			},
		},
		{
			name: "redirect with regex and named captures",
			fbRedirects: []Redirect{
				{Regex: `^/p/(?P<id>\d+)$`, Destination: "/post/:id"},
			},
			want: []NginxRedirect{
				{Pattern: `^/p/(?P<id>\d+)$`, Target: "/post/$id", Code: 301},
			},
		},
		{
			name: "missing destination",
			fbRedirects: []Redirect{
				{Source: "/foo"},
			},
			wantErr: true,
		},
		{
			name: "invalid glob (unbalanced group)",
			fbRedirects: []Redirect{
				{Source: "/foo@(bar", Destination: "/target"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TranslateRedirects(tc.fbRedirects)
			if (err != nil) != tc.wantErr {
				t.Fatalf("TranslateRedirects() returned error %v, wantErr %t", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TranslateRedirects() returned unexpected diff (-want +got):\\n%s", diff)
			}
		})
	}
}

func TestTranslateRewrites(t *testing.T) {
	tests := []struct {
		name       string
		fbRewrites []Rewrite
		want       []NginxRewrite
		wantErr    bool
	}{
		{
			name: "basic rewrite",
			fbRewrites: []Rewrite{
				{Source: "/foo", Destination: "/bar"},
			},
			want: []NginxRewrite{
				{Pattern: `^/foo$`, Target: "/bar"},
			},
		},
		{
			name: "glob rewrite",
			fbRewrites: []Rewrite{
				{Source: "/blog/**", Destination: "/news"},
			},
			want: []NginxRewrite{
				{Pattern: `^/blog(?:/.*)?$`, Target: "/news"},
			},
		},
		{
			name: "segment rewrite (splat)",
			fbRewrites: []Rewrite{
				{Source: "/library-seg/:splat*", Destination: "/archive/:splat*"},
			},
			want: []NginxRewrite{
				{Pattern: `^/library-seg/(?P<splat>.+)$`, Target: "/archive/$splat"},
			},
		},
		{
			name: "regex rewrite",
			fbRewrites: []Rewrite{
				{Regex: `^/p/(.*)$`, Destination: "/post/$1"},
			},
			want: []NginxRewrite{
				{Pattern: `^/p/(.*)$`, Target: "/post/$1"},
			},
		},
		{
			name: "rewrite with colon segments",
			fbRewrites: []Rewrite{
				{Source: "/blog/:slug", Destination: "/blogs/:slug"},
			},
			want: []NginxRewrite{
				{Pattern: `^/blog/(?P<slug>[^/]+)$`, Target: "/blogs/$slug"},
			},
		},
		{
			name: "rewrite with regex and numeric captures",
			fbRewrites: []Rewrite{
				{Regex: `^/p/(\d+)$`, Destination: "/post/:1"},
			},
			want: []NginxRewrite{
				{Pattern: `^/p/(\d+)$`, Target: "/post/$1"},
			},
		},
		{
			name: "hosting redirect (ignored in rewrite translation)",
			fbRewrites: []Rewrite{
				{Source: "/foo"},
			},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TranslateRewrites(tc.fbRewrites)
			if (err != nil) != tc.wantErr {
				t.Fatalf("TranslateRewrites() returned error %v, wantErr %t", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TranslateRewrites() returned unexpected diff (-want +got):\\n%s", diff)
			}
		})
	}
}
