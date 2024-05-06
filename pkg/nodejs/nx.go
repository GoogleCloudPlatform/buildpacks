package nodejs

import (
	"encoding/json"
	"io/ioutil"
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

// NxProjectJSON represents the contents of a project.json file.
type NxProjectJSON struct {
	Name        string    `json:"name"`
	ProjectType string    `json:"projectType"`
	Prefix      string    `json:"prefix"`
	SourceRoot  string    `json:"sourceRoot"`
	Targets     NxTargets `json:"targets"`
}

// ReadNxProjectJSONIfExists returns deserialized nx.json from the given dir. If the provided dir
// does not contain a nx.json file it returns nil. Empty dir string uses the current working
// directory.
func ReadNxProjectJSONIfExists(dir string) (*NxProjectJSON, error) {
	f := filepath.Join(dir, "project.json")
	raw, err := ioutil.ReadFile(f)
	if os.IsNotExist(err) {
		// Return an empty struct if the file doesn't exist (null object pattern).
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
