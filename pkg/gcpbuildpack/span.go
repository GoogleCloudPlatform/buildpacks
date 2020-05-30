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
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Status is the standard Google API status.
type Status int

// Statuses matching https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto All error status are attributed to the user, except for those denoted.
const (
	StatusOk                 Status = 0
	StatusCancelled          Status = 1
	StatusUnknown            Status = 2
	StatusInvalidArgument    Status = 3
	StatusDeadlineExceeded   Status = 4
	StatusNotFound           Status = 5
	StatusAlreadyExists      Status = 6
	StatusPermissionDenied   Status = 7
	StatusResourceExhausted  Status = 8
	StatusFailedPrecondition Status = 9
	StatusAborted            Status = 10
	StatusOutOfRange         Status = 11
	StatusUnimplemented      Status = 12
	StatusInternal           Status = 13 // Google-attributed error
	StatusUnavailable        Status = 14
	StatusDataLoss           Status = 15
	StatusUnauthenticated    Status = 16
)

func (s Status) String() string {
	return []string{
		"OK",
		"CANCELLED",
		"UNKNOWN",
		"INVALID_ARGUMENT",
		"DEADLINE_EXCEEDED",
		"NOT_FOUND",
		"ALREADY_EXISTS",
		"PERMISSION_DENIED",
		"RESOURCE_EXHAUSTED",
		"FAILED_PRECONDITION",
		"ABORTED",
		"OUT_OF_RANGE",
		"UNIMPLEMENTED",
		"INTERNAL",
		"UNAVAILABLE",
		"DATA_LOSS",
		"UNAUTHENTICATED",
	}[s]
}

var fromStatusString = map[string]Status{
	"OK":                  StatusOk,
	"CANCELLED":           StatusCancelled,
	"UNKNOWN":             StatusUnknown,
	"INVALID_ARGUMENT":    StatusInvalidArgument,
	"DEADLINE_EXCEEDED":   StatusDeadlineExceeded,
	"NOT_FOUND":           StatusNotFound,
	"ALREADY_EXISTS":      StatusAlreadyExists,
	"PERMISSION_DENIED":   StatusPermissionDenied,
	"RESOURCE_EXHAUSTED":  StatusResourceExhausted,
	"FAILED_PRECONDITION": StatusFailedPrecondition,
	"ABORTED":             StatusAborted,
	"OUT_OF_RANGE":        StatusOutOfRange,
	"UNIMPLEMENTED":       StatusUnimplemented,
	"INTERNAL":            StatusInternal,
	"UNAVAILABLE":         StatusUnavailable,
	"DATA_LOSS":           StatusDataLoss,
	"UNAUTHENTICATED":     StatusUnauthenticated,
}

// MarshalJSON marshals the enum as a quoted json string
func (s Status) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", s)), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *Status) UnmarshalJSON(b []byte) error {
	var val string
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	st, ok := fromStatusString[strings.ToUpper(val)]
	if !ok {
		return fmt.Errorf("unknown value %q", val)
	}
	*s = st
	return nil
}

type spanInfo struct {
	name       string
	start      time.Time
	end        time.Time
	attributes map[string]interface{}
	status     Status
}

func newSpanInfo(name string, start, end time.Time, attributes map[string]interface{}, status Status) (*spanInfo, error) {
	if name == "" {
		return nil, fmt.Errorf("span name required")
	}
	if start.After(end) {
		return nil, fmt.Errorf("start is after end")
	}

	// TODO: validate attributes
	// See https://cloud.google.com/trace/docs/reference/v2/rest/v2/Attributes

	return &spanInfo{
		name:       name,
		start:      start,
		end:        end,
		attributes: attributes,
		status:     status,
	}, nil
}

func (ctx *Context) createSpanName(cmd []string) string {
	var trimmed []string
	for _, c := range cmd {
		t := strings.TrimSpace(c)
		if t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return fmt.Sprintf("Exec %q", strings.Join(trimmed, " "))
}
