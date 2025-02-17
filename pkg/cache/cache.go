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
	"io/ioutil"

	"github.com/buildpacks/libcnb/v2"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

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
			b, err := ioutil.ReadFile(f)
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

	prevHash := ctx.GetMetadata(l, key)
	ctx.Debugf("Current dependency hash: %q", currHash)
	ctx.Debugf("  Cache dependency hash: %q", prevHash)

	if prevHash == "" {
		ctx.Debugf("No cache metadata found from a previous build for key: %q, skipping cache.", key)
	}

	cached := currHash == prevHash
	if cached {
		ctx.CacheHit(l.Name)
	} else {
		ctx.CacheMiss(l.Name)
	}
	return currHash, cached, nil
}
