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
	"encoding/xml"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// AppEngineWebXMLApp represents the appengine-web.xml file.
type AppEngineWebXMLApp struct {
	XMLName         xml.Name `xml:"appengine-web-app"`
	Entrypoint      string   `xml:"entrypoint"`
	AppEngineAPIs   bool     `xml:"app-engine-apis"`
	Runtime         string   `xml:"runtime"`
	SessionsEnabled bool     `xml:"sessions-enabled"`
}

// ParseAppEngineWebXML unmarshals the provided appengine-web.xml into a AppEngineWebXMLApp.
func ParseAppEngineWebXML(xmlFile []byte) (*AppEngineWebXMLApp, error) {
	var AppEngineWebXMLApp AppEngineWebXMLApp
	if err := xml.Unmarshal(xmlFile, &AppEngineWebXMLApp); err != nil {
		return nil, gcp.UserErrorf("parsing appengine-web.xml: %v", err)
	}

	return &AppEngineWebXMLApp, nil
}
