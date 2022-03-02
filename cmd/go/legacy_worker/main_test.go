// Copyright 2021 Google LLC
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

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	buildpacktest "github.com/GoogleCloudPlatform/buildpacks/internal/buildpacktest"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestDetect(t *testing.T) {
	testCases := []struct {
		name  string
		files map[string]string
		env   []string
		stack string
		want  int
	}{
		{
			name: "with target",
			env:  []string{"GOOGLE_FUNCTION_TARGET=HelloWorld"},
			want: 0,
		},
		{
			name: "without target",
			want: 100,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buildpacktest.TestDetectWithStack(t, detectFn, tc.name, tc.files, tc.env, tc.stack, tc.want)
		})
	}
}

func TestCreateMainGoFile(t *testing.T) {
	workerTmplFile = `
	import (
		"log"
		"net/http"
		userfunction "{{ .Package }}"
	)

	func extraUseOfUserFunction() {
		var handler interface{} = userfunction.{{ .Target }}
	}

	func main() {
		var handler interface{} = userfunction.{{ .Target }}
		http.HandleFunc("/", extraUseOfUserFunction)
		err := http.ListenAndServe(":"+"8080", nil)
		if err != nil {
			log.Fatalf("Error starting the Worker server for Go: %s\n", err)
		}
	}
	`
	const want = `
	import (
		"log"
		"net/http"
		userfunction "example.com/user/package/name"
	)

	func extraUseOfUserFunction() {
		var handler interface{} = userfunction.UserFunctionName
	}

	func main() {
		var handler interface{} = userfunction.UserFunctionName
		http.HandleFunc("/", extraUseOfUserFunction)
		err := http.ListenAndServe(":"+"8080", nil)
		if err != nil {
			log.Fatalf("Error starting the Worker server for Go: %s\n", err)
		}
	}
	`

	tmpDir := t.TempDir()
	buildpackWorkerRoot := filepath.Join(tmpDir, "converter", "worker")
	if err := os.MkdirAll(buildpackWorkerRoot, os.ModePerm); err != nil {
		t.Fatalf("error creating temp directory at %q: %s", buildpackWorkerRoot, err)
	}

	ctx := gcp.NewContext()
	info := fnInfo{
		Target:  "UserFunctionName",
		Package: "example.com/user/package/name",
	}

	targetMain := filepath.Join(tmpDir, "main.go")
	if err := createMainGoFile(ctx, info, targetMain); err != nil {
		t.Fatalf("error creating file at path %q from worker template: %s", targetMain, err)
	}

	b, err := ioutil.ReadFile(targetMain)
	if err != nil {
		t.Fatalf("error reading file at path %q: %s", targetMain, err)
	}

	got := string(b)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("generated main.go mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateMainGoModFile(t *testing.T) {
	// gcp_buildpacks/cmd/go/legacy_worker/converter/worker/gomod.tmpl
	const want = `// Copyright 2021 Google LLC
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

module functions.local/app

require fnmod v0.0.0

replace fnmod => ./serverless_function_source_code`

	tmpDir := t.TempDir()
	ctx := gcp.NewContext()

	targetGoMod := filepath.Join(tmpDir, "go.mod")
	if err := createMainGoModFile(ctx, "fnmod", targetGoMod); err != nil {
		t.Fatalf("error creating file at path %q from go.mod template: %s", targetGoMod, err)
	}

	b, err := ioutil.ReadFile(targetGoMod)
	if err != nil {
		t.Fatalf("error reading file at path %q: %s", targetGoMod, err)
	}

	got := string(b)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("generated go.mod mismatch (-want +got):\n%s", diff)
	}
}
