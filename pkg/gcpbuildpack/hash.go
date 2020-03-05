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

package gcpbuildpack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
)

// HashFileContents stores the name of the file whose contents should be hashed.
type HashFileContents string

// DependencyHash returns the hash of the language version and application files.
func DependencyHash(ctx *Context, langVer string, files ...string) (string, error) {
	components := []interface{}{langVer}
	for _, file := range files {
		components = append(components, HashFileContents(file))
	}
	return computeSHA256(ctx, components...)
}

// computeSHA256 returns a sha256 string based on the given components.
func computeSHA256(ctx *Context, components ...interface{}) (result string, err error) {
	h := sha256.New()

	h.Write([]byte(ctx.BuildpackID()))
	h.Write([]byte(ctx.BuildpackVersion()))

	for _, c := range components {
		switch val := c.(type) {
		case bool:
			h.Write([]byte(strconv.FormatBool(val)))
		case int:
			h.Write([]byte(strconv.Itoa(val)))
		case string:
			h.Write([]byte(val))
		case HashFileContents:
			fname := string(val)
			f, err := os.Open(fname)
			if err != nil {
				return "", fmt.Errorf("opening %q: %v", fname, err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					err = fmt.Errorf("closing %q: %v", fname, err)
				}
			}()
			if _, err := io.Copy(h, f); err != nil {
				return "", fmt.Errorf("reading %q: %v", fname, err)
			}
		default:
			return "", fmt.Errorf("unknown type %T", val)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
