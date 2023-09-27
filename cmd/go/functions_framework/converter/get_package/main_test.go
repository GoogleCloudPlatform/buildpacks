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

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtract(t *testing.T) {
	tcs := []struct {
		name  string
		files map[string]string
		want  *parsedPackage
	}{
		{
			name: "one package",
			files: map[string]string{
				"fn.go": `package cloudeventfunction

import (
	"context"
	"log"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func init() {
	functions.CloudEvent("HelloStorage", HelloStorage)
}

func HelloStorage(ctx context.Context, e cloudevents.Event) error {
	log.Printf("Storage trigger in Go in GCF Gen 2!, %v!", e)
	return nil
}`,
				"go.mod": `module example.com/cloudeventfunction

go 1.13

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.4.1
	github.com/cloudevents/sdk-go/v2 v2.2.0
)`,
			},
			want: &parsedPackage{
				Name: "cloudeventfunction",
				Imports: map[string]struct{}{
					"context": struct{}{},
					"github.com/GoogleCloudPlatform/functions-framework-go/functions": struct{}{},
					"github.com/cloudevents/sdk-go/v2":                                struct{}{},
					"log":                                                             struct{}{},
				},
			},
		}, {
			name: "one package with two files",
			files: map[string]string{
				"fn.go": `package cloudeventfunction

import (
	"context"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func HelloStorage(ctx context.Context, e cloudevents.Event) error {
	log.Printf("Storage trigger in Go in GCF Gen 2!, %v!", e)
	return nil
}`,
				"init.go": `package cloudeventfunction

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.CloudEvent("HelloStorage", HelloStorage)
}`,
			},
			want: &parsedPackage{
				Name: "cloudeventfunction",
				Imports: map[string]struct{}{
					"context": struct{}{},
					"github.com/GoogleCloudPlatform/functions-framework-go/functions": struct{}{},
					"github.com/cloudevents/sdk-go/v2":                                struct{}{},
					"log":                                                             struct{}{},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "golang_bp_test")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Fatalf("removing temp dir: %v", err)
				}
			}()

			for f, c := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(c), 0644); err != nil {
					t.Fatalf("writing file %s: %v", f, err)
				}
			}

			got, err := extract(dir)
			if err != nil {
				t.Fatalf("error extracting package data from Go application: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractFailures(t *testing.T) {
	tcs := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "two packages",
			files: map[string]string{
				"foo.go": `package foo`,
				"bar.go": `package bar`,
			},
		}, {
			name: "bad file",
			files: map[string]string{
				"foo.go": `not a go file`,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "golang_bp_test")
			if err != nil {
				t.Fatalf("creating temp dir: %v", err)
			}
			defer func() {
				err = os.RemoveAll(dir)
				if err != nil {
					t.Fatalf("removing temp dir: %v", err)
				}
			}()

			for f, c := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(c), 0644); err != nil {
					t.Fatalf("writing file %s: %v", f, err)
				}
			}

			if _, err := extract(dir); err == nil {
				t.Fatalf("expected Extract() error, got nil")
			}
		})
	}
}

func TestMarshalUnmarshalPackage(t *testing.T) {
	pkgObj := &parsedPackage{
		Name: "httpfunction",
		Imports: map[string]struct{}{
			"fmt":      struct{}{},
			"net/http": struct{}{},
		},
	}

	pkgJSON := `{"name":"httpfunction","imports":{"fmt":{},"net/http":{}}}`

	b, err := json.Marshal(pkgObj)
	if err != nil {
		t.Fatalf("error marshaling Package to JSON, Package: %v, error: %v", pkgObj, err)
	}

	gotJSON := string(b)
	if gotJSON != pkgJSON {
		t.Errorf("JSON mismatch, got: %q, want: %q", gotJSON, pkgJSON)
	}

	var gotPkg *parsedPackage
	if err := json.Unmarshal([]byte(gotJSON), &gotPkg); err != nil {
		t.Errorf("error unmarshaling Package from JSON, JSON: %q, error: %v", gotJSON, err)
	}

	if diff := cmp.Diff(pkgObj, gotPkg); diff != "" {
		t.Errorf("Package{} mismatch (-want +got):\n%s", diff)
	}
}
