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

const mainTextTemplateDeclarative = `// Binary main file implements an HTTP server that loads and runs user's code
// on incoming HTTP requests.
// As this file must compile statically alongside the user code, this file
// will be copied into the function image and the '{{.Package}}' and
// package name derived from the 'FUNCTION_TARGET' env var. That edited file
// will then be compiled as with the user's function code to produce an
// executable app binary that launches the HTTP server.
package main

import (
	"log"
	"net/http"
	"os"

	// Blank import the function package to run the init(), which is where
	// declarative function signatures are expected to be registered.
	_ "{{.Package}}"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {
	// Don't invoke the function for reserved URLs.
	http.HandleFunc("/robots.txt", http.NotFound)
	http.HandleFunc("/favicon.ico", http.NotFound)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("Function failed to start: %v\n", err)
	}
}`
