// Copyright 2023 Google LLC
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

package publisher

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"gopkg.in/yaml.v2"
)

var (
	appHostingYAMLPath string = testdata.MustGetPath("testdata/apphosting.yaml")
	bundleYAMLPath     string = testdata.MustGetPath("testdata/bundle.yaml")
)

func yamlStringToStruct(yamlString string) (backendResources, error) {
	var structData backendResources
	err := yaml.Unmarshal([]byte(yamlString), &structData)
	if err != nil {
		return backendResources{}, err
	}

	return structData, nil
}

func TestPublisher(t *testing.T) {
	testCases := []struct {
		desc string
	}{
		{"find combined output yaml file contents"},
	}

	for _, test := range testCases {
		res, _ := Publish(appHostingYAMLPath, bundleYAMLPath)
		YAMLData, err := yamlStringToStruct(res)

		if err != nil {
			t.Errorf("not able to convert the YAML to a struct in testcase %v. Error was %v", test.desc, err)
		}
		if YAMLData.CPU != 3 {
			t.Errorf("Publisher(%v, %v) = %v, want cpu: 3", appHostingYAMLPath, bundleYAMLPath, YAMLData)
		}

	}
}
