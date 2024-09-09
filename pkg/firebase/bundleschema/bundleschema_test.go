package bundleschema

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/testdata"
	"github.com/google/go-cmp/cmp"
)

func TestReadAndValidateFromFile(t *testing.T) {
	testCases := []struct {
		desc             string
		inputBundleYAML  string
		wantBundleSchema BundleSchema
		wantErr          bool
	}{
		{
			desc:            "Read properly formatted bundle yaml schema properly",
			inputBundleYAML: testdata.MustGetPath("testdata/bundle_valid.yaml"),
			wantBundleSchema: BundleSchema{
				Env: []EnvironmentVariable{
					EnvironmentVariable{Variable: "SSR_PORT", Value: "8080", Availability: []string{"RUNTIME"}},
					EnvironmentVariable{Variable: "HOSTNAME", Value: "0.0.0.0", Availability: []string{"RUNTIME"}},
				},
			},
		},
		{
			desc:            "Throw an error when the file doesn't exist",
			inputBundleYAML: testdata.MustGetPath("testdata/nonexistant.yaml"),
			wantErr:         true,
		},
		{
			desc:            "Throw an error when an env field contains a secret",
			inputBundleYAML: testdata.MustGetPath("testdata/bundle_invalidenv_secret.yaml"),
			wantErr:         true,
		},
		{
			desc:            "Throw an error when an env field does not contain a value",
			inputBundleYAML: testdata.MustGetPath("testdata/bundle_invalidenv_value.yaml"),
			wantErr:         true,
		},
		{
			desc:            "Throw an error when an env field contains an invalid availability value",
			inputBundleYAML: testdata.MustGetPath("testdata/bundle_invalidenv_availability.yaml"),
			wantErr:         true,
		},
	}

	for _, test := range testCases {
		s, err := ReadAndValidateFromFile(test.inputBundleYAML)

		if !test.wantErr {
			if err != nil {
				t.Errorf("unexpected error for ReadAndValidateFromFile(%q): %v", test.desc, err)
			}

			if diff := cmp.Diff(test.wantBundleSchema, s); diff != "" {
				t.Errorf("unexpected YAML for test %q, (-want, +got):\n%v", test.desc, diff)
			}

		} else {
			if err == nil {
				t.Errorf("ReadAndValidateFromFile(%q) = %v, want error", test.desc, err)
			}
		}
	}
}
