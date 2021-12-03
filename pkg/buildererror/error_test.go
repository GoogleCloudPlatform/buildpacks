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

package buildererror

import (
	"testing"
)

func TestGenerateErrorId(t *testing.T) {
	result1 := GenerateErrorID("abc", "def")
	if len(result1) != errorIDLength {
		t.Fatalf("len errorId got %d, want %d", len(result1), errorIDLength)
	}

	result2 := GenerateErrorID("abc")
	if result2 == result1 {
		t.Errorf("error IDs are not unique to different inputs")
	}
}
