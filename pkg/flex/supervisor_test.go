package flex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appyaml"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestSupervisorConfFiles(t *testing.T) {
	testCases := []struct {
		name       string
		rc         appyaml.RuntimeConfig
		want       SupervisorFiles
		writeFiles []string
	}{
		{
			name: "supervisor files are not provided",
			rc:   appyaml.RuntimeConfig{},
			want: SupervisorFiles{
				AddSupervisorConf:       defaultAddSupervisorConf,
				SupervisorConf:          defaultSupervisorConf,
				SupervisorConfExists:    false,
				AddSupervisorConfExists: false},
		},
		{
			name: "changed supervisor files but they don't exist",
			rc:   appyaml.RuntimeConfig{SupervisordConfAddition: "add.conf", SupervisordConfOverride: "override.conf"},
			want: SupervisorFiles{
				AddSupervisorConf:       "add.conf",
				SupervisorConf:          "override.conf",
				SupervisorConfExists:    false,
				AddSupervisorConfExists: false,
			},
		},
		{
			name: "changed supervisor files and they exist",
			rc:   appyaml.RuntimeConfig{SupervisordConfAddition: "add.conf", SupervisordConfOverride: "override.conf"},
			want: SupervisorFiles{
				AddSupervisorConf:       "add.conf",
				SupervisorConf:          "override.conf",
				SupervisorConfExists:    true,
				AddSupervisorConfExists: true,
			},
			writeFiles: []string{"add.conf", "override.conf"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := gcp.NewContext(gcp.WithApplicationRoot(dir))
			for _, file := range tc.writeFiles {
				os.Create(filepath.Join(dir, file))
			}

			defer os.RemoveAll(dir)

			gotFiles, err := SupervisorConfFiles(ctx, tc.rc, ctx.ApplicationRoot())
			if err != nil {
				t.Fatalf("SupervisorConfFiles returns error: %v", err)
			}

			if diff := cmp.Diff(tc.want, gotFiles); diff != "" {
				t.Errorf("SupervisorConfFiles(, %v) returns unexpected struct(-want, +got):\n%s", tc.rc, diff)
			}
		})
	}
}
