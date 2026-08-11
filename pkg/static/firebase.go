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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"text/template"
)

// FirebaseNginxMap represents an Nginx map directive for Firebase custom headers.
type FirebaseNginxMap struct {
	VariableName string
	DefaultValue string
	Mappings     []FirebaseNginxMapping
}

// FirebaseNginxMapping represents a single mapping in an Nginx map for Firebase custom headers.
type FirebaseNginxMapping struct {
	Regex string
	Value string
}

// FirebaseNginxHeader represents an Nginx add_header directive for Firebase custom headers.
type FirebaseNginxHeader struct {
	Key          string
	VariableName string
}

// FirebaseNginxConfigParams holds the runtime configuration parameters for templating nginx.conf in the Firebase buildpack.
type FirebaseNginxConfigParams struct {
	RootPath      string
	MimeTypesPath string
	Maps          []FirebaseNginxMap
	Headers       []FirebaseNginxHeader
	Redirects     []NginxRedirect
	Rewrites      []NginxRewrite
}

const firebaseNginxConfTmpl = `
pid /tmp/nginx.pid;
error_log /dev/stderr notice;

events {
    worker_connections 1024;
}

http {
    include {{.MimeTypesPath}};
    access_log /dev/stdout;

    client_body_temp_path /tmp/nginx_client_body;
    proxy_temp_path /tmp/nginx_proxy;
    fastcgi_temp_path /tmp/nginx_fastcgi;
    uwsgi_temp_path /tmp/nginx_uwsgi;
    scgi_temp_path /tmp/nginx_scgi;

    # Define a variable for literal dollar sign to avoid interpolation
    geo $literal_dollar {
        default "$";
    }

{{range .Maps}}
    map $uri ${{.VariableName}} {
        default {{.DefaultValue}};
{{range .Mappings}}        "~{{.Regex}}" {{.Value}};
{{end}}    }
{{end}}
    server {
        listen 8080;
        root {{.RootPath}};
        index index.html;
{{range .Headers}}
        add_header {{.Key}} ${{.VariableName}} always;{{end}}

        {{range .Redirects}}
        location ~ {{.Pattern}} {
            return {{.Code}} {{.Target}};
        }
        {{end}}

        {{range .Rewrites}}
        location ~ {{.Pattern}} {
            rewrite {{.Pattern}} {{.Target}} break;
        }
        {{end}}

        # Default Fallback
        location / {
            try_files $uri $uri/ /index.html;
        }

        absolute_redirect off;
    }
}
`

// WriteFirebaseNginxConfig compiles the Firebase Nginx configuration template with parameters and writes it to disk.
func WriteFirebaseNginxConfig(dstPath string, params FirebaseNginxConfigParams) error {
	tmpl, err := template.New(NginxConfFile).Parse(firebaseNginxConfTmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, params)
}

// HeaderConfig represents a single header key-value pair in firebase.json.
type HeaderConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Header represents the headers configuration in firebase.json.
type Header struct {
	Source  string         `json:"source"`
	Regex   string         `json:"regex,omitempty"`
	Headers []HeaderConfig `json:"headers"`
}

// Run represents a Cloud Run configuration for rewrites.
type Run struct {
	ServiceID string `json:"serviceId"`
	Region    string `json:"region,omitempty"`
}

// Rewrite represents a rewrite rule in firebase.json.
type Rewrite struct {
	Source       string `json:"source"`
	Regex        string `json:"regex,omitempty"`
	Destination  string `json:"destination,omitempty"`
	Function     string `json:"function,omitempty"`
	Run          *Run   `json:"run,omitempty"`
	DynamicLinks bool   `json:"dynamicLinks,omitempty"`
}

// Redirect represents a redirect rule in firebase.json.
type Redirect struct {
	Source      string `json:"source"`
	Regex       string `json:"regex,omitempty"`
	Destination string `json:"destination"`
	Type        int    `json:"type"`
}

// HostingConfig represents a single hosting target configuration in firebase.json.
type HostingConfig struct {
	Target        string     `json:"target,omitempty"`
	Site          string     `json:"site,omitempty"`
	Public        string     `json:"public,omitempty"`
	CleanUrls     bool       `json:"cleanUrls,omitempty"`
	TrailingSlash *bool      `json:"trailingSlash,omitempty"`
	Rewrites      []Rewrite  `json:"rewrites,omitempty"`
	Redirects     []Redirect `json:"redirects,omitempty"`
	Headers       []Header   `json:"headers,omitempty"`
}

// FirebaseJSON represents the root structure of firebase.json.
type FirebaseJSON struct {
	Hosting []HostingConfig `json:"-"`
}

// UnmarshalJSON is a custom unmarshaler to handle the hosting field as either an object or an array.
func (f *FirebaseJSON) UnmarshalJSON(data []byte) error {
	var raw struct {
		Hosting json.RawMessage `json:"hosting"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw.Hosting) == 0 || string(raw.Hosting) == "null" {
		return nil
	}

	// Try unmarshaling as an array
	var arr []HostingConfig
	if err := json.Unmarshal(raw.Hosting, &arr); err == nil {
		f.Hosting = arr
		return nil
	}

	// Try unmarshaling as a single object
	var single HostingConfig
	if err := json.Unmarshal(raw.Hosting, &single); err != nil {
		return fmt.Errorf("failed to parse hosting config: %w", err)
	}
	f.Hosting = []HostingConfig{single}
	return nil
}

// ParseFirebaseConfig reads and parses a firebase.json file.
// If the file does not exist, it returns an empty slice and no error.
func ParseFirebaseConfig(path string) ([]HostingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No firebase.json found
		}
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var fb FirebaseJSON
	if err := json.Unmarshal(data, &fb); err != nil {
		return nil, fmt.Errorf("parsing firebase.json: %w", err)
	}

	return fb.Hosting, nil
}

func normalizeVarName(key string) string {
	var sb strings.Builder
	sb.WriteString("hdr_")
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func escapeNginxValue(val string) string {
	val = strings.ReplaceAll(val, "\\", "\\\\")
	val = strings.ReplaceAll(val, "\"", "\\\"")
	val = strings.ReplaceAll(val, "$", "${literal_dollar}")
	return `"` + val + `"`
}

func isMultiValueHeader(key string) bool {
	return key == "Set-Cookie" || key == "Link"
}

type headerSlot struct {
	key          string
	variableName string
	mappings     []FirebaseNginxMapping
}

type headerRegexKey struct {
	header string
	regex  string
}

// PrepareNginxHeaders converts Firebase header configs into Nginx map and header structures.
func PrepareNginxHeaders(config *HostingConfig) ([]FirebaseNginxMap, []FirebaseNginxHeader, error) {
	if config == nil || len(config.Headers) == 0 {
		return nil, nil, nil
	}

	singleValueSlots, multiValueSlots, err := processHeaderSlots(config)
	if err != nil {
		return nil, nil, err
	}

	finalSlots := orderedSingleValueSlots(config, singleValueSlots)
	finalSlots = append(finalSlots, multiValueSlots...)

	maps, headers := buildNginxDirectives(finalSlots)
	return maps, headers, nil
}

func processHeaderSlots(config *HostingConfig) (map[string]*headerSlot, []headerSlot, error) {
	singleValueSlots := make(map[string]*headerSlot)
	var multiValueSlots []headerSlot
	multiValueCounts := make(map[string]int)
	seen := make(map[headerRegexKey]bool)

	// Process in reverse order so the last match in firebase.json wins (first match in Nginx map)
	for _, h := range slices.Backward(config.Headers) {
		if err := processHeaderRule(h, singleValueSlots, &multiValueSlots, multiValueCounts, seen); err != nil {
			return nil, nil, err
		}
	}
	return singleValueSlots, multiValueSlots, nil
}

func processHeaderRule(h Header, singleValueSlots map[string]*headerSlot, multiValueSlots *[]headerSlot, multiValueCounts map[string]int, seen map[headerRegexKey]bool) error {
	regex, err := GlobToRegex(h.Source)
	if err != nil {
		return fmt.Errorf("invalid source pattern %q: %w", h.Source, err)
	}

	// Process headers in reverse order as well to respect last-match-wins within the same block
	for _, kv := range slices.Backward(h.Headers) {
		key := http.CanonicalHeaderKey(kv.Key)
		val := kv.Value

		if !isMultiValueHeader(key) {
			refKey := headerRegexKey{header: key, regex: regex}
			if seen[refKey] {
				continue
			}
			seen[refKey] = true
			addSingleValueMapping(key, val, regex, singleValueSlots)
		} else {
			addMultiValueMapping(key, val, regex, multiValueSlots, multiValueCounts)
		}
	}
	return nil
}

func addSingleValueMapping(key, val, regex string, singleValueSlots map[string]*headerSlot) {
	slot, exists := singleValueSlots[key]
	if !exists {
		slot = &headerSlot{
			key:          key,
			variableName: normalizeVarName(key),
		}
		singleValueSlots[key] = slot
	}
	slot.mappings = append(slot.mappings, FirebaseNginxMapping{
		Regex: regex,
		Value: escapeNginxValue(val),
	})
}

func addMultiValueMapping(key, val, regex string, multiValueSlots *[]headerSlot, multiValueCounts map[string]int) {
	idx := multiValueCounts[key]
	multiValueCounts[key]++
	varName := fmt.Sprintf("%s_%d", normalizeVarName(key), idx)

	*multiValueSlots = append(*multiValueSlots, headerSlot{
		key:          key,
		variableName: varName,
		mappings: []FirebaseNginxMapping{
			{
				Regex: regex,
				Value: escapeNginxValue(val),
			},
		},
	})
}

func orderedSingleValueSlots(config *HostingConfig, singleValueSlots map[string]*headerSlot) []headerSlot {
	var finalSlots []headerSlot
	keySet := make(map[string]bool)
	for _, h := range config.Headers {
		for _, kv := range h.Headers {
			canonicalKey := http.CanonicalHeaderKey(kv.Key)
			if !keySet[canonicalKey] && !isMultiValueHeader(canonicalKey) {
				keySet[canonicalKey] = true
				if slot, exists := singleValueSlots[canonicalKey]; exists {
					finalSlots = append(finalSlots, *slot)
				}
			}
		}
	}
	return finalSlots
}

func buildNginxDirectives(slots []headerSlot) ([]FirebaseNginxMap, []FirebaseNginxHeader) {
	var maps []FirebaseNginxMap
	var headers []FirebaseNginxHeader

	for _, slot := range slots {
		maps = append(maps, FirebaseNginxMap{
			VariableName: slot.variableName,
			DefaultValue: `""`,
			Mappings:     slot.mappings,
		})
		headers = append(headers, FirebaseNginxHeader{
			Key:          slot.key,
			VariableName: slot.variableName,
		})
	}
	return maps, headers
}
