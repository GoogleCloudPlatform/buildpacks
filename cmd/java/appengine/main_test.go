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

package main

import (
	"strings"
	"testing"
)

func TestHasMainTrue(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name: "simple case",
			manifestContents: `Main-Class: test
another: example`,
		},
		{
			name: "with line continuation",
			// The manifest spec states that a continuation line must end with a trailing space.
			manifestContents: `simple: example 
 wrapping
Main-Class: example`,
		},
		{
			name: "main-class with line continuation",
			manifestContents: `simple: example 
 wrapping
Main-Class: example 
 line wrap
New-Entry: example`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.manifestContents)

			if !hasMain(r) {
				t.Errorf("hasMain() returned false, wanted true")
			}
		})
	}
}

func TestHasMainFalse(t *testing.T) {
	testCases := []struct {
		name             string
		manifestContents string
	}{
		{
			name: "no main class entry",
			manifestContents: `Not-Main-Class: test
another: example`,
		},
		{
			name: "main class with preceding space",
			manifestContents: `simple: example
 wrapping
 Main-Class: example`,
		},
		{
			name:             "main class with no entry",
			manifestContents: `Main-Class: `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.manifestContents)

			if hasMain(r) {
				t.Errorf("hasMain() returned true, wanted false")
			}
		})
	}
}
