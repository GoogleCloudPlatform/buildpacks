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

// Package main tests building source that has an empty go.mod file.
package main

import (
	"fmt"
	"net/http"
	"runtime"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "PASS")
}

func version(w http.ResponseWriter, r *http.Request) {
	wants, ok := r.URL.Query()["want"]
	if !ok || len(wants) != 1 || wants[0] == "" {
		fmt.Fprintf(w, "FAIL: ?want must be set to a version")
		return
	}
	got := runtime.Version()
	want := fmt.Sprintf("go%s", wants[0])
	if got != want {
		fmt.Fprintf(w, "FAIL: current version: %s; want %s", got, want)
	} else {
		fmt.Fprintf(w, "PASS")
	}
}

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/version", version)
	http.ListenAndServe(":8080", nil)
}
