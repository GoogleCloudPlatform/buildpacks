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

// Package main tests building source that has a go.mod file without gomod dependencies.
// It also tests compatiblity with the golang/appengine package.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"rsc.io/quote"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if got, want := quote.Hello(), "Hello, world."; got != want {
		fmt.Fprintf(w, "FAIL: quote.Hello() = %s, want %s\n", got, want)
		return
	}

	// Check that the path at compile time includes /srv/. This is done for compatibility with the
	// appengine package, which requires it for historical reasons. See the go/appengine_gomod
	// buildpack.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(w, "FAIL: runtime.Caller(0) not ok")
		return
	}
	sep := string(filepath.Separator)
	srv := sep + "srv" + sep
	if !strings.Contains(file, srv) {
		fmt.Fprintf(w, "FAIL: caller file path %s does not contain %s\n", file, srv)
		return
	}
	fmt.Fprintln(w, "PASS")
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
