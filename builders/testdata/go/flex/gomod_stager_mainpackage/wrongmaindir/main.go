// Copyright 2023 Google LLC
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

// Package main tests building source that has a go.mod file with only a go-app-stager defined main path.
// This file should NOT be used by the acceptance tests -- it exists to provide
// a second buildable unit in the same test. Here is what we want to test: the
// stager should provide a _main-package-path, and the unified builder should
// respect it. However, the go build buildpack gets confused if you don't use
// _main-package-path and there is more than one buildable in the same folder.
// Therefore, we're testing whether the unified builder respects the stager.
package main

import (
	"fmt"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Incorrectly using the main package specified by go-app-stager")
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
