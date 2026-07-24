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
	"os"
)

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
