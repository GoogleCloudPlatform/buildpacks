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

package nodejs

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestReadPackageJSON(t *testing.T) {
	want := PackageJSON{
		Engines: packageEnginesJSON{
			Node: "my-node",
			NPM:  "my-npm",
		},
		Scripts: packageScriptsJSON{
			Start: "my-start",
		},
		Dependencies: map[string]string{
			"a": "1.0",
			"b": "2.0",
		},
		DevDependencies: map[string]string{
			"c": "3.0",
		},
	}

	got, err := ReadPackageJSON(testdata.MustGetPath("testdata/test-read-package/"))
	if err != nil {
		t.Fatalf("ReadPackageJSON got error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadPackageJSON\ngot %#v\nwant %#v", *got, want)
	}
}
