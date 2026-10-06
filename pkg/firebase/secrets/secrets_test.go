package secrets

import (
	"context"
	"hash/crc32"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/fakesecretmanager"
	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	"github.com/google/go-cmp/cmp"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	ctx                  context.Context = context.Background()
	pinnedSecretName     string          = "projects/test-project/secrets/secretID/versions/5"
	latestSecretName     string          = "projects/test-project/secrets/secretID/versions/latest"
	secretString         string          = "secretString"
	secretStringChecksum int64           = int64(crc32.Checksum([]byte(secretString), crc32.MakeTable(crc32.Castagnoli)))
)

func TestNormalize(t *testing.T) {
	testCases := []struct {
		desc      string
		projectID string
		inputEnv  []apphostingschema.EnvironmentVariable
		wantEnv   []apphostingschema.EnvironmentVariable
		wantErr   string
	}{
		{
			desc:      "Normalize different secret formats properly",
			projectID: "test-project",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_ONE", Secret: "secretID", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_TWO", Secret: "secretID@5", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_THREE", Secret: "projects/test-project/secrets/secretID", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_FOUR", Secret: "projects/test-project/secrets/secretID/versions/6", Availability: []string{"BUILD"}},
			},
			wantEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_ONE", Secret: "projects/test-project/secrets/secretID/versions/latest", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_TWO", Secret: "projects/test-project/secrets/secretID/versions/5", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_THREE", Secret: "projects/test-project/secrets/secretID/versions/latest", Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "SECRET_FORMAT_FOUR", Secret: "projects/test-project/secrets/secretID/versions/6", Availability: []string{"BUILD"}},
			},
		},
		{
			desc:      "Change nothing when the env section is empty",
			projectID: "test-project",
			inputEnv:  nil,
			wantEnv:   nil,
		},
		{
			desc:      "Throw an error when secret name is improperly formatted",
			projectID: "test-project",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "INVALID_SECRET_FORMAT", Secret: "secretID@@5", Availability: []string{"BUILD"}},
			},
			wantErr: "Improper Secret Format",
		},
	}

	for _, test := range testCases {
		err := Normalize(test.inputEnv, test.projectID)

		// Happy Case
		if test.wantErr == "" {
			if err != nil {
				t.Errorf("Normalize(%q) = %v, want %v", test.desc, err, test.wantEnv)
			}

			if diff := cmp.Diff(test.wantEnv, test.inputEnv); diff != "" {
				t.Errorf("unexpected normalized envVars for test %q (+got, -want):\n%v", test.desc, diff)
			}
		} else {
			// Error Case
			if err == nil {
				t.Errorf("calling Normalize did not produce an error for test %q", test.desc)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error not in expected format for test %q.\nGot: %v\nWant: %v", test.desc, err, test.wantErr)
			}
		}
	}
}

func TestPinVersions(t *testing.T) {
	testCases := []struct {
		desc     string
		inputEnv []apphostingschema.EnvironmentVariable
		wantEnv  []apphostingschema.EnvironmentVariable
		wantErr  bool
	}{
		{
			desc: "Pin secret values properly",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "PINNED_SECRET", Secret: pinnedSecretName, Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "LATEST_SECRET", Secret: latestSecretName, Availability: []string{"BUILD"}},
			},
			wantEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "PINNED_SECRET", Secret: pinnedSecretName, Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "LATEST_SECRET", Secret: pinnedSecretName, Availability: []string{"BUILD"}},
			},
		},
		{
			desc:     "Change nothing when the env section is empty",
			inputEnv: nil,
			wantEnv:  nil,
		},
		{
			desc: "Throw an error when secret version is not found",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "LATEST_SECRET_INVALID_FORMAT", Secret: "projects/test-project/secrets/invalidSecretID/versions/latest", Availability: []string{"BUILD"}},
			},
			wantErr: true,
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		SecretVersionResponses: map[string]fakesecretmanager.GetSecretVersionResponse{
			latestSecretName: fakesecretmanager.GetSecretVersionResponse{
				SecretVersion: &smpb.SecretVersion{
					Name:  pinnedSecretName,
					State: smpb.SecretVersion_ENABLED,
				},
			},
		},
	}

	for _, test := range testCases {
		err := PinVersions(ctx, fakeSecretClient, test.inputEnv)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("PinVersions(%q) = %v, want %v", test.desc, err, test.wantEnv)
			}

			if diff := cmp.Diff(test.wantEnv, test.inputEnv); diff != "" {
				t.Errorf("unexpected pinned envVars for test %q (+got, -want):\n%v", test.desc, diff)
			}
			// Error Path
		} else {
			if err == nil {
				t.Errorf("PinVersions(%q) = %v, want error", test.desc, err)
			}
		}
	}
}

func TestGenerateBuildDereferencedEnvMap(t *testing.T) {
	testCases := []struct {
		desc        string
		inputEnv    []apphostingschema.EnvironmentVariable
		wantEnvVars map[string]string
		wantErr     bool
	}{
		{
			desc: "Dereference secret values properly, stripping any non-build environment variables",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "API_URL", Value: "api.service.com", Availability: []string{"BUILD", "RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "API_URL_ONLY_RUNTIME", Value: "api.service.com", Availability: []string{"RUNTIME"}},
				apphostingschema.EnvironmentVariable{Variable: "PINNED_SECRET", Secret: pinnedSecretName, Availability: []string{"BUILD"}},
				apphostingschema.EnvironmentVariable{Variable: "PINNED_SECRET_ONLY_RUNTIME", Secret: pinnedSecretName, Availability: []string{"RUNTIME"}},
			},
			wantEnvVars: map[string]string{
				"API_URL":       "api.service.com",
				"PINNED_SECRET": secretString,
			},
		},
		{
			desc:        "Return an empty map when the env section is missing",
			inputEnv:    nil,
			wantEnvVars: map[string]string{},
		},
		{
			desc: "Throw an error when secret version is not found",
			inputEnv: []apphostingschema.EnvironmentVariable{
				apphostingschema.EnvironmentVariable{Variable: "LATEST_SECRET_INVALID_FORMAT", Secret: "projects/test-project/secrets/invalidSecretID/versions/latest", Availability: []string{"BUILD", "RUNTIME"}},
			},
			wantErr: true,
		},
	}

	fakeSecretClient := &fakesecretmanager.FakeSecretClient{
		AccessSecretVersionResponses: map[string]fakesecretmanager.AccessSecretVersionResponse{
			pinnedSecretName: fakesecretmanager.AccessSecretVersionResponse{
				Response: &smpb.AccessSecretVersionResponse{
					Payload: &smpb.SecretPayload{
						Data:       []byte(secretString),
						DataCrc32C: &secretStringChecksum,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		gotEnvVars, err := GenerateBuildDereferencedEnvMap(ctx, fakeSecretClient, test.inputEnv)

		// Happy Path
		if !test.wantErr {
			if err != nil {
				t.Errorf("unexpected error for DereferenceSecrets(%q): %v", test.desc, err)
			}

			if diff := cmp.Diff(test.wantEnvVars, gotEnvVars); diff != "" {
				t.Errorf("unexpected dereferenced secrets for test %q (+got, -want):\n%v", test.desc, diff)
			}
		} else {
			if err == nil {
				t.Errorf("DereferenceSecrets(%q) = %v, want error", test.desc, err)
			}
		}
	}
}
