package nodejs

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestReadNxJSONIfExists(t *testing.T) {
	want := NxJSON{
		DefaultProject:     "default-project",
		NxCloudAccessToken: "access-token",
	}

	got, err := ReadNxJSONIfExists(testdata.MustGetPath("testdata/test-read-nx-project/"))
	if err != nil {
		t.Fatalf("ReadNxJSONIfExists got error: %v", err)
	}
	if got == nil {
		t.Fatalf("ReadNxJSONIfExists did not find nx.json")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadNxJSONIfExists\ngot %#v\nwant %#v", *got, want)
	}
}

func TestReadProjectJSONIfExists(t *testing.T) {
	want := NxProjectJSON{
		Name:        "nx-app",
		ProjectType: "application",
		Prefix:      "test-read-nx-project",
		SourceRoot:  "apps/nx-app/src",
		Targets: NxTargets{
			Build: NxBuild{
				Executor: "@framework/builder",
			},
		},
	}

	got, err := ReadNxProjectJSONIfExists(testdata.MustGetPath("testdata/test-read-nx-project/apps/nx-app/"))
	if err != nil {
		t.Fatalf("ReadNxProjectJSONIfExists got error: %v", err)
	}
	if got == nil {
		t.Fatalf("ReadNxProjectJSONIfExists did not find project.json")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadNxProjectJSONIfExists\ngot %#v\nwant %#v", *got, want)
	}
}
