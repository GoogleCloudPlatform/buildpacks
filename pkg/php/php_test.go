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
)

func TestReadComposerJSON(t *testing.T) {
	d, err := ioutil.TempDir("/tmp", "test-read-composer-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(d)

	contents := strings.TrimSpace(`
{
  "scripts": {
    "gcp-build": "my-script"
  }
}
`)

	if err := ioutil.WriteFile(filepath.Join(d, "composer.json"), []byte(contents), 0644); err != nil {
		t.Fatalf("Failed to write composer.json: %v", err)
	}

	want := ComposerJSON{
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
