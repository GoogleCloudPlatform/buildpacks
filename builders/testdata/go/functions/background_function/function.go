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

// Package myfunc tests a background function.
package myfunc

import (
	"context"
	"fmt"

	"cloud.google.com/go/functions/metadata"
)

// Func is a background function.
func Func(ctx context.Context, data map[string]interface{}) error {
	meta, err := metadata.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("metadata.FromContext failed: %v", err)
	}

	// spot check metadata is non-empty
	if meta.EventID == "" {
		return fmt.Errorf("metadata missing event ID, got metadata: %v", meta)
	}

	if len(data) == 0 {
		return fmt.Errorf("data parameter was empty, expected non-empty data")
	}

	return nil
}
