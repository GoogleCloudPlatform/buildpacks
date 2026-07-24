package nodejs

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
)

func TestReadTurboJSONIfExists(t *testing.T) {
	want := TurboJSON{
		Tasks: TurboTasks{
			Build: TurboBuild{
				Outputs: []string{"dist/**", ".next/**"},
				Cache:   true,
			},
		},
	}

	got, err := ReadTurboJSONIfExists(testdata.MustGetPath("testdata/test-read-turbo-project/"))
	if err != nil {
		t.Fatalf("ReadTurboJSONIfExists got error: %v", err)
	}
	if got == nil {
		t.Fatalf("ReadTurboJSONIfExists did not find turbo.json")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("ReadTurboJSONIfExists\ngot %#v\nwant %#v", *got, want)
	}
}
