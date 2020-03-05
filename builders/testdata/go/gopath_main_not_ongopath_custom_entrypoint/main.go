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

// Package main tests building source that has no dependencies but expects a flag that can only be specified with a custom entrypoint
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	passFlag string
)

func init() {
	flag.StringVar(&passFlag, "passflag", "", "\"PASS\" string used to confirm that a flag was successfully passed through a custom entrypoint")
	flag.Parse()
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, passFlag)
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
