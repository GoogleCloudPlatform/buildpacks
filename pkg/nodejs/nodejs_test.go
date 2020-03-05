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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadPackageJSON(t *testing.T) {
	d, err := ioutil.TempDir("/tmp", "test-read-package-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(d)

	contents := strings.TrimSpace(`
{
  "engines": {
    "node": "my-node"
  },
  "scripts": {
    "start": "my-start"
  },
	"dependencies": {
	  "a": "1.0",
		"b": "2.0"
	},
	"devDependencies": {
	  "c": "3.0"
	}
}
`)

	if err := ioutil.WriteFile(filepath.Join(d, "package.json"), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	want := PackageJSON{
		Engines: packageEnginesJSON{
			Node: "my-node",
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
	got, err := ReadPackageJSON(d)
	if err != nil {
		t.Errorf("ReadPackageJSON got error: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadPackageJSON\ngot %#v\nwant %#v", *got, want)
	}
}
