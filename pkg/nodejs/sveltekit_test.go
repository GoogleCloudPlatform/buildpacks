package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"google3/security/safeopen/safeopen"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-cmp/cmp"
)

func TestSvelteKitStartCommand(t *testing.T) {
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
			name:         "sveltekit app",
			buildExists:  true,
			configExists: true,
			want:         []string{"node", "build/index.js"},
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if tc.configExists {
				_, err := safeopen.CreateBeneath(home, "svelte.config.js")
				if err != nil {
					t.Fatalf("failed to create server.js: %v", err)
				}
			}
			if tc.buildExists {
				dir := filepath.Join(home, "build")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create build directory: %v", err)
				}
				_, err := safeopen.CreateBeneath(dir, "index.js")
				if err != nil {
					t.Fatalf("failed to create index.js: %v", err)
				}
			}
			ctx := gcpbuildpack.NewContext(gcpbuildpack.WithApplicationRoot(home))

			got, err := SvelteKitStartCommand(ctx)
			if err != nil {
				t.Fatalf("SvelteKitStartCommand() got error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("SvelteKitStartCommand() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDetectSvelteKitAutoAdapter(t *testing.T) {
	testsCases := []struct {
		name string
		pjs  string
		want bool
	}{
		{
			name: "with sveltekit auto adapter",
			pjs: `{
					"devDependencies": {
						"@sveltejs/adapter-auto": "^3.0.0"
					}
				}`,
			want: true,
		},
		{
			name: "with two sveltekit adapters",
			pjs: `{
					"devDependencies": {
						"@sveltejs/adapter-auto": "^3.0.0",
						"@sveltejs/adapter-node": "^5.0.1"
					}
				}`,
			want: false,
		},
		{
			name: "with no sveltekit adapters",
			pjs: `{
        "scripts": {
					"dev": "vite dev",
					"build": "vite build",
					"preview": "vite preview",
					"check": "svelte-kit sync && svelte-check --tsconfig ./jsconfig.json",
					"check:watch": "svelte-kit sync && svelte-check --tsconfig ./jsconfig.json --watch"
				}
			}`,
			want: false,
		},
		{
			name: "no scripts",
			pjs:  `{}`,
			want: false,
		},
	}
	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			var pjs *PackageJSON = nil
			if tc.pjs != "" {
				if err := json.Unmarshal([]byte(tc.pjs), &pjs); err != nil {
					t.Fatalf("failed to unmarshal package.json: %s, error: %v", tc.pjs, err)
				}
			}

			if got := DetectSvelteKitAutoAdapter(pjs); got != tc.want {
				t.Errorf("DetectSvelteKitAutoAdapter() = %v, want %v", got, tc.want)
			}
		})
	}
}
