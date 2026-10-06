package gcpbuildpack

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	"github.com/buildpacks/libcnb/v2"
)

func TestCacheLayer(t *testing.T) {
	testCases := []struct {
		name    string
		noCache string
		want    bool
	}{
		{
			name: "no env",
			want: true,
		},
		{
			name:    "env falsy",
			noCache: "0",
			want:    true,
		},
		{
			name:    "env truthy",
			noCache: "1",
			want:    false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.noCache != "" {
				t.Setenv(env.NoCache, tc.noCache)
			}
			ctx := NewContext()
			l := libcnb.Layer{Name: "test"}
			if err := CacheLayer(ctx, &l); err != nil {
				t.Fatalf("CacheLayer() unexpected error: %v", err)
			}
			if l.Cache != tc.want {
				t.Errorf("CacheLayer() %v=%q got %v, want %v", env.NoCache, tc.noCache, l.Cache, tc.want)
			}
		})
	}
}
