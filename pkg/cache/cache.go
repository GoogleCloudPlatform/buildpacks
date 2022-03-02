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
	"io/ioutil"

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

// Hash creates a sha256 hash from the given cache options.
func Hash(ctx *gcp.Context, opts ...Option) (result string, err error) {
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

	return hex.EncodeToString(h.Sum(nil)), nil
}
