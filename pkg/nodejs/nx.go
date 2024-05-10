package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// NxTargets specifies configs for Nx execution targets.
type NxTargets struct {
	Build NxBuild `json:"build"`
}

// NxBuild specifies the structure of an Nx build target config.
type NxBuild struct {
	Executor string `json:"executor"`
}

// NxJSON represents the contents of a nx.json file.
// See https://nx.dev/reference/nx-json for documentation on the configuration file schema.
type NxJSON struct {
	DefaultProject     string `json:"defaultProject"`
	NxCloudAccessToken string `json:"nxCloudAccessToken"`
}

// NxProjectJSON represents the contents of a project.json file.
type NxProjectJSON struct {
	Name        string    `json:"name"`
	ProjectType string    `json:"projectType"`
	Prefix      string    `json:"prefix"`
	SourceRoot  string    `json:"sourceRoot"`
	Targets     NxTargets `json:"targets"`
}

// ReadNxJSONIfExists returns deserialized nx.json from the given dir. If the provided dir
// does not contain a nx.json file it returns nil. Empty dir string uses the current working
// directory.
func ReadNxJSONIfExists(dir string) (*NxJSON, error) {
	f := filepath.Join(dir, "nx.json")
	raw, err := os.ReadFile(f)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %v", f, err)
	}

	var nxJSON NxJSON
	if err := json.Unmarshal(raw, &nxJSON); err != nil {
		return nil, gcp.UserErrorf("unmarshalling %s: %v", f, err)
	}
	return &nxJSON, nil
}

// ReadNxProjectJSONIfExists returns deserialized project.json from the given dir. If the provided
// dir does not contain a project.json file it returns nil. Empty dir string uses the current
// working directory.
func ReadNxProjectJSONIfExists(dir string) (*NxProjectJSON, error) {
	f := filepath.Join(dir, "project.json")
	raw, err := os.ReadFile(f)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, gcp.InternalErrorf("reading %s: %v", f, err)
	}

	var pjson NxProjectJSON
	if err := json.Unmarshal(raw, &pjson); err != nil {
		return nil, gcp.UserErrorf("unmarshalling %s: %v", f, err)
	}
	return &pjson, nil
}
