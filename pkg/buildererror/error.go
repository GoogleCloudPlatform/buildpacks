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

// Package buildererror provides an interface for builder Errors.
package buildererror

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

const (
	errorIDLength = 8
)

// ID is a short error code passed to the user for supportability.
type ID string

// Error is a gcpbuildpack structured error.
type Error struct {
	BuildpackID      string `json:"buildpackId"`
	BuildpackVersion string `json:"buildpackVersion"`
	Type             Status `json:"errorType"`
	Status           Status `json:"canonicalCode"`
	ID               ID     `json:"errorId"`
	Message          string `json:"errorMessage"`
	internalError    error  `json:"-"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("(error ID: %s):\n%s", e.ID, e.Message)
}

func (e *Error) Unwrap() error {
	return e.internalError
}

// Errorf constructs an Error.
func Errorf(status Status, format string, args ...any) *Error {
	err := fmt.Errorf(format, args...)
	return &Error{
		Type:          status,
		Status:        status,
		ID:            GenerateErrorID(err.Error()),
		Message:       err.Error(),
		internalError: err,
	}
}

// InternalErrorf constructs an Error with status StatusInternal (Google-attributed SLO).
func InternalErrorf(format string, args ...any) *Error {
	return Errorf(StatusInternal, format, args...)
}

// UserErrorf constructs an Error with status StatusUnknown (user-attributed SLO).
func UserErrorf(format string, args ...any) *Error {
	return Errorf(StatusUnknown, format, args...)
}

// GenerateErrorID creates a short hash from the provided parts.
func GenerateErrorID(parts ...string) ID {
	h := sha256.New()
	for _, p := range parts {
		io.WriteString(h, p)
	}
	result := fmt.Sprintf("%x", h.Sum(nil))

	// Since this is only a reporting aid for support, we truncate the hash to make it more human friendly.
	return ID(strings.ToLower(result[:errorIDLength]))
}
