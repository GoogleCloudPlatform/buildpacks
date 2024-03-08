package apphostingschema

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

func int32Ptr(i int) *int32 {
	v := new(int32)
	*v = int32(i)
	return v
}

func float32Ptr(i int32) *float32 {
	v := new(float32)
	*v = float32(i)
	return v
}

func TestReadAndValidateAppHostingSchemaFromFile(t *testing.T) {
	testCases := []struct {
		desc                 string
		inputAppHostingYAML  string
		wantAppHostingSchema AppHostingSchema
		wantErr              bool
	}{
		{
			desc:                "Read properly formatted app hosting YAML schema properly",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_valid.yaml"),
			wantAppHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          float32Ptr(3),
					MemoryMiB:    int32Ptr(1024),
					Concurrency:  int32Ptr(100),
					MaxInstances: int32Ptr(4),
				},
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "STORAGE_BUCKET", Value: "mybucket.appspot.com", Availability: []string{"BUILD", "BACKEND"}},
					EnvironmentVariable{Variable: "API_KEY", Secret: "myApiKeySecret", Availability: []string{"BUILD"}},
					EnvironmentVariable{Variable: "PINNED_API_KEY", Secret: "myApiKeySecret@5"}},
			},
		},
		{
			desc:                "Read YAML schema missing an env section properly",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_missingenv.yaml"),
			wantAppHostingSchema: AppHostingSchema{
				RunConfig: RunConfig{
					CPU:          float32Ptr(3),
					MemoryMiB:    int32Ptr(1024),
					Concurrency:  int32Ptr(100),
					MaxInstances: int32Ptr(4),
				},
			},
		},
		{
			desc:                 "Return an empty schema when the file doesn't exist",
			inputAppHostingYAML:  testdata.MustGetPath("testdata/nonexistant.yaml"), // File doesn't exist
			wantAppHostingSchema: AppHostingSchema{},
		},
		{
			desc:                "Throw an error when an env field contains both a value and a secret",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidenv_valuesecret.yaml"),
			wantErr:             true,
		},
		{
			desc:                "Throw an error when an env field contains an invalid availability value",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidenv_availability.yaml"),
			wantErr:             true,
		},
		{
			desc:                "Throw an error when a run config field contains an invalid value",
			inputAppHostingYAML: testdata.MustGetPath("testdata/apphosting_invalidrunconfig.yaml"),
			wantErr:             true,
		},
	}

	for _, test := range testCases {
		s, err := ReadAndValidateAppHostingSchemaFromFile(test.inputAppHostingYAML)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("unexpected error for ReadAppHostingSchemaFromFile(%q): %v", test.desc, err)
			}

			if diff := cmp.Diff(test.wantAppHostingSchema, s); diff != "" {
				t.Errorf("unexpected YAML for test %q, (+got, -want):\n%v", test.desc, diff)
			}

			// Error Path
		} else {
			if err == nil {
				t.Errorf("ReadAppHostingSchemaFromFile(%q) = %v, want error", test.desc, err)
			}
		}
	}
}
