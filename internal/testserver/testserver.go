// Copyright 2022 Google LLC
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

// Package testserver provides utility functions for stubbing HTTP requests in tests.
package testserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type config struct {
	httpStatus   int
	responseFile string
	responseJSON string
	mockURL      *string
}

// Option configures test servers.
type Option func(o *config)

// WithStatus sets the http response code to return.
func WithStatus(httpStatus int) Option {
	return func(c *config) {
		c.httpStatus = httpStatus
	}
}

// WithJSON sets the JSON payload the server send in the response body.
func WithJSON(json string) Option {
	return func(c *config) {
		c.responseJSON = json
	}
}

// WithFile sets the path of a file the server should send as a response.
func WithFile(path string) Option {
	return func(c *config) {
		c.responseFile = path
	}
}

// WithMockURL stubs the provided URL to point to this test server for the duration of a unit test.
func WithMockURL(url *string) Option {
	return func(c *config) {
		c.mockURL = url
	}
}

// New creates and starts a test server with the provided configurations and returns its URL.
func New(t *testing.T, opts ...Option) *httptest.Server {
	t.Helper()
	options := config{}
	for _, o := range opts {
		o(&options)
	}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if options.httpStatus != 0 {
			w.WriteHeader(options.httpStatus)
		}
		if options.responseFile != "" {
			http.ServeFile(w, r, options.responseFile)
			return
		}
		if options.responseJSON != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		if _, err := w.Write([]byte(options.responseJSON)); err != nil {
			// Not using Fatalf because this runs in a separate Go Routine.
			t.Errorf("sending stubbed http response: %v", err)
		}
	}))
	t.Cleanup(svr.Close)

	if options.mockURL != nil {
		origVal := *options.mockURL
		t.Cleanup(func() {
			*options.mockURL = origVal
		})
		*options.mockURL = svr.URL + "?p1=%s&p2=%s&p3=%s"
	}
	return svr
}
