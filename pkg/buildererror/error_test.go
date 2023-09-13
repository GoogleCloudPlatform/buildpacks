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
	"errors"
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

type externalErrorType struct {
	msg string
}

func (e externalErrorType) Error() string {
	return e.msg
}

// TestWrappedErrors tests that the custom error type works with wrapped errors and the %w printing
// directive. The %w directive is only supported by fmt.Errorf and is used to wrap errors with more
// context while still preserving the inner errors message and type. In other contexts, %w is the
// same as %v and will print a struct instead of a string.64
func TestWrappedErrors(t *testing.T) {
	externalError := externalErrorType{msg: "external error"}
	innerError := InternalErrorf("inner error: %w", externalError)
	outerError := InternalErrorf("outer error: %w", innerError)

	got := outerError.Error()
	want := "(error ID: 189e479d):\nouter error: (error ID: 028d0ed8):\ninner error: external error"

	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(outerError, externalError) {
		t.Errorf("errors.Is() did not match type: %#v", externalErrorType{})
	}

	var placeholder externalErrorType
	if !errors.As(outerError, &placeholder) {
		t.Errorf("errors.As() did not match type: %#v", externalErrorType{})
	}
}
