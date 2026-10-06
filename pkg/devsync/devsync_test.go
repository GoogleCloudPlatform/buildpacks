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

package devsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateAppRunScript(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("creating app dir: %v", err)
	}

	templatePath := filepath.Join(appDir, "run.template")
	runPath := filepath.Join(appDir, "run")

	templateContent := "#!/bin/bash\ncd /workspace || exit 1\nexec chpst -e ./env -P {{ENTRYPOINT}}\n"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("writing run.template: %v", err)
	}

	envVars := map[string]string{
		"PYTHONPATH": "/custom/path",
	}

	if err := UpdateAppRunScript(dir, "node --watch server.js", envVars); err != nil {
		t.Fatalf("UpdateAppRunScript failed: %v", err)
	}

	b, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatalf("reading app/run: %v", err)
	}

	want := "#!/bin/bash\ncd /workspace || exit 1\nexec chpst -e ./env -P node --watch server.js\n"
	if string(b) != want {
		t.Errorf("app/run content = %q, want %q", string(b), want)
	}

	// Verify env var file creation
	envVal, err := os.ReadFile(filepath.Join(appDir, "env", "PYTHONPATH"))
	if err != nil {
		t.Fatalf("reading app/env/PYTHONPATH: %v", err)
	}
	if string(envVal) != "/custom/path" {
		t.Errorf("app/env/PYTHONPATH = %q, want %q", string(envVal), "/custom/path")
	}

	// Verify run.template is preserved
	if _, err := os.Stat(templatePath); err != nil {
		t.Errorf("run.template was not preserved: %v", err)
	}
}
