// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cache implements functions to generate cache keys.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb/v2"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetadata"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	analyzedPathEnv     = "CNB_ANALYZED_PATH"
	defaultAnalyzedPath = "/layers/analyzed.toml"
)

// analyzedTOML represents the relevant metadata schema of CNB analyzed.toml.
type analyzedTOML struct {
	Metadata struct {
		Buildpacks []analyzedBuildpack `toml:"buildpacks"`
	} `toml:"metadata"`
	Buildpacks []analyzedBuildpack `toml:"buildpacks"`
}

type analyzedBuildpack struct {
	ID     string         `toml:"id"`
	Key    string         `toml:"key"`
	Layers map[string]any `toml:"layers"`
}

func findKeyInMap(v any, targetKey string) string {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if k == targetKey {
				if s, ok := item.(string); ok {
					return s
				}
			}
			if s := findKeyInMap(item, targetKey); s != "" {
				return s
			}
		}
	case map[string]string:
		for k, s := range val {
			if k == targetKey {
				return s
			}
		}
	}
	return ""
}

// readHashFromAnalyzed attempts to read the cached dependency hash for the given buildpack and
// key from the remote OCI metadata file (analyzed.toml). Returns an empty string if analyzed.toml
// does not exist, is unreadable, or does not contain the key for the buildpack to keep the logic
// fail-open.
func readHashFromAnalyzed(ctx *gcp.Context, key string) string {
	path := os.Getenv(analyzedPathEnv)
	if path == "" {
		path = defaultAnalyzedPath
	}
	ctx.Debugf("Checking for analyzed.toml at path: %s", path)

	var analyzed analyzedTOML
	if _, err := toml.DecodeFile(path, &analyzed); err != nil {
		ctx.Debugf("Could not decode analyzed.toml at %s: %v", path, err)
		return ""
	}

	bpID := ctx.BuildpackID()
	ctx.Debugf("Searching analyzed.toml for buildpack %q, layer key %q...", bpID, key)

	bps := analyzed.Metadata.Buildpacks
	if len(bps) == 0 {
		bps = analyzed.Buildpacks
	}

	for _, bp := range bps {
		if bp.ID != bpID && bp.Key != bpID {
			continue
		}
		for _, layer := range bp.Layers {
			if s := findKeyInMap(layer, key); s != "" {
				ctx.Debugf("Found remote OCI metadata hash in analyzed.toml for %s/%s: %s", bpID, key, s)
				return s
			}
		}
	}
	ctx.Debugf("Key %q not found for buildpack %q in analyzed.toml.", key, bpID)
	return ""
}

// Option is a function that returns strings to be hashed when computing a cache key.
type Option func() ([]string, error)

// WithStrings returns a cache option for string values.
func WithStrings(strings ...string) Option {
	return func() ([]string, error) {
		return strings, nil
	}
}

// WithFiles returns a cache option that hashes contents of the files. Callers can
// detect if a file did not exist by checking returned error values against
// os.IsNotFound(...).
func WithFiles(files ...string) Option {
	return func() ([]string, error) {
		var strings []string
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			strings = append(strings, string(b))
		}
		return strings, nil
	}
}

// hash creates a sha256 hash from the given cache options.
func hash(ctx *gcp.Context, opts ...Option) (string, error) {
	h := sha256.New()

	h.Write([]byte(ctx.BuildpackID()))
	h.Write([]byte(ctx.BuildpackVersion()))

	for _, opt := range opts {
		strings, err := opt()
		if err != nil {
			return "", err
		}
		for _, s := range strings {
			h.Write([]byte(s))
		}
	}

	hash := hex.EncodeToString(h.Sum(nil))
	return hash, nil
}

// Add adds the key-value to the cache for the given layer for future builds.
func Add(ctx *gcp.Context, l *libcnb.Layer, key string, value string) {
	ctx.SetMetadata(l, key, value)
}

// HashAndCheck computes a hash value according to the cache options provided and checks if there is
// a cache hit or miss by looking at the provided layer; returns the computed hash and if there
// was a cache.
func HashAndCheck(ctx *gcp.Context, l *libcnb.Layer, key string, opts ...Option) (string, bool, error) {
	currHash, err := hash(ctx, opts...)
	if err != nil {
		return "", false, fmt.Errorf("computing dependency hash: %w", err)
	}

	prevHashOnDisk := ctx.GetMetadata(l, key)
	prevHashRemote := readHashFromAnalyzed(ctx, key)

	ctx.Debugf("Current dependency hash: %q", currHash)
	ctx.Debugf("Disk cache dependency hash: %q", prevHashOnDisk)
	ctx.Debugf("Remote analyzed.toml dependency hash: %q", prevHashRemote)

	// Record telemetry metrics based on remote OCI analyzed.toml and local disk metadata.
	if (prevHashRemote != "" && currHash == prevHashRemote) || (prevHashOnDisk != "" && currHash == prevHashOnDisk) {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.LayerCacheHitCounterID).Increment(1)
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.LayerCacheStatus, "hit")
	} else if prevHashRemote != "" || prevHashOnDisk != "" {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.LayerCacheMissCounterID).Increment(1)
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.LayerCacheStatus, "miss")
	} else {
		buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.LayerCacheColdCounterID).Increment(1)
		buildermetadata.GlobalBuilderMetadata().SetValue(buildermetadata.LayerCacheStatus, "cold")
	}

	// Execution caching behavior: strictly preserve existing disk-based caching behavior.
	cached := prevHashOnDisk != "" && currHash == prevHashOnDisk
	if cached {
		ctx.CacheHit(l.Name)
	} else {
		ctx.CacheMiss(l.Name)
	}
	return currHash, cached, nil
}
