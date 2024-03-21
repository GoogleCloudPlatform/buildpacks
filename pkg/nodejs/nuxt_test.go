package nodejs

import (
	"os"
	"path/filepath"
	"testing"

	"google3/security/safeopen/safeopen"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestNuxtStartCommand(t *testing.T) {
	testsCases := []struct {
		name         string
		configExists bool
		buildExists  bool
		want         []string
	}{
		{
			name: "no config or build",
			want: nil,
		},
		{
			name:        "no config",
			buildExists: true,
			want:        nil,
		},
		{
			name:         "no build",
			configExists: true,
			want:         nil,
		},
		{
			name:         "nuxt app",
			buildExists:  true,
			configExists: true,
			want:         []string{"node", ".output/server/index.mjs"},
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.configExists {
				_, err := safeopen.CreateBeneath(home, "nuxt.config.ts")
				if err != nil {
					t.Fatalf("failed to create server.js: %v", err)
				}
			}
			if tc.buildExists {
				dir := filepath.Join(home, ".output/server/")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create server directory: %v", err)
				}
				_, err := safeopen.CreateBeneath(dir, "index.mjs")
				if err != nil {
					t.Fatalf("failed to create index.mjs: %v", err)
				}
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(home))

			got, err := NuxtStartCommand(ctx)
			if err != nil {
				t.Fatalf("NuxtStartCommand() got error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("NuxtStartCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
