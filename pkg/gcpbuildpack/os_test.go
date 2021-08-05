// Copyright 2021 Google LLC
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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIsWritable(t *testing.T) {
	ctx, cleanUp := simpleContext(t)
	defer cleanUp()

	temp := t.TempDir()

	testCases := []struct {
		mode os.FileMode
		want bool
	}{
		{0000, false},
		{0001, false},
		{0002, false},
		{0003, false},
		{0004, false},
		{0005, false},
		{0006, false},
		{0007, false},
		{0010, false},
		{0020, false},
		{0030, false},
		{0040, false},
		{0050, false},
		{0060, false},
		{0070, false},
		{0100, false},
		{0200, true},
		{0300, true},
		{0400, false},
		{0500, false},
		{0600, true},
		{0700, true},
		{0777, true},
	}
	for _, tc := range testCases {
		t.Run(tc.mode.String(), func(t *testing.T) {
			f, err := os.Create(filepath.Join(temp, fmt.Sprintf("file_%v", tc.mode)))
			if err != nil {
				t.Fatalf("os.Create(): %v", err)
			}

			if err := f.Chmod(tc.mode); err != nil {
				t.Fatalf("f.Chmod(%s): %v", tc.mode, err)
			}

			if got, want := ctx.IsWritable(f.Name()), tc.want; got != want {
				t.Errorf("gcp.IsWritable(%s) = %t, want %t", f.Name(), got, want)
			}
		})
	}
}
