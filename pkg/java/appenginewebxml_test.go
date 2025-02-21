// Copyright 2025 Google LLC
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

package java

import (
	"embed"
	"encoding/xml"
	"testing"

	"github.com/google/go-cmp/cmp"
)

//go:embed testdata/*
var testDataWebXML embed.FS

func TestParseValidAppEngineWebXML(t *testing.T) {
	tests := []struct {
		path string
		want AppEngineWebXMLApp
	}{
		{
			path: "testdata/appengine-web.xml",
			want: AppEngineWebXMLApp{
				XMLName:         xml.Name{Space: "http://appengine.google.com/ns/1.0", Local: "appengine-web-app"},
				Entrypoint:      "java -cp myapp.jar com.example.MyMain",
				AppEngineAPIs:   true,
				Runtime:         "java17",
				SessionsEnabled: true,
			},
		},
		{
			path: "testdata/minimal_appengine-web.xml",
			want: AppEngineWebXMLApp{
				XMLName:         xml.Name{Space: "http://appengine.google.com/ns/1.0", Local: "appengine-web-app"},
				Runtime:         "java21",
				AppEngineAPIs:   true,
				SessionsEnabled: false,
			},
		},
		{
			path: "testdata/extra_config_appengine-web.xml",
			want: AppEngineWebXMLApp{
				XMLName:         xml.Name{Space: "http://appengine.google.com/ns/1.0", Local: "appengine-web-app"},
				Runtime:         "java11",
				SessionsEnabled: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			xmlFile, err := testData.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("Unable to find appengine-web.xml file %s, %v", tc.path, err)
			}

			got, err := ParseAppEngineWebXML(xmlFile)
			if err != nil {
				t.Fatalf("ParseAppEngineWebXML failed to parse appengine-web.xml: %v", err)
			}

			if !cmp.Equal(*got, tc.want) {
				t.Errorf("ParseAppEngineWebXML\ngot %#v\nwant %#v", *got, tc.want)
			}
		})
	}
}
