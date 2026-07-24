// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// TurboTasks specifies config options for Turbo tasks.
type TurboTasks struct {
	Build TurboBuild `json:"build"`
}

// TurboBuild specifies configs for the Turbo build task.
type TurboBuild struct {
	Outputs []string `json:"outputs,omitempty"` // the list of output files to cache
	Cache   bool     `json:"cache,omitempty"`   // whether cache is enabled
}

// TurboJSON represents the contents of a turbo.json file.
// See https://turborepo.com/docs/reference/configuration for documentation on the configuration file schema.
type TurboJSON struct {
	Tasks TurboTasks `json:"tasks"`
}

// ReadTurboJSONIfExists returns deserialized turbo.json from the given dir. If the provided dir
// does not contain a turbo.json file it returns nil. Empty dir string uses the current working
// directory.
func ReadTurboJSONIfExists(dir string) (*TurboJSON, error) {
	f := filepath.Join(dir, "turbo.json")
	raw, err := os.ReadFile(f)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %v", f, err)
	}

	var turboJSON TurboJSON
	if err := json.Unmarshal(raw, &turboJSON); err != nil {
		return nil, gcp.UserErrorf("unmarshalling %s: %v", f, err)
	}
	return &turboJSON, nil
}
