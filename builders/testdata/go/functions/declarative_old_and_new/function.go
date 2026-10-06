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

// Package myfunc tests code that uses both the old and new registration
// libraries.
package myfunc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"rsc.io/quote"
)

func init() {
	ctx := context.Background()
	funcframework.RegisterHTTPFunctionContext(ctx, "/", Func)
	functions.HTTP("Func", myFunc) // FUNCTION_TARGET=Func should call this one
	functions.HTTP("NotTargetFunc", Func)
}

// myFunc is a test function.
func myFunc(w http.ResponseWriter, r *http.Request) {
	if quote.Hello() == "Hello, world." {
		fmt.Fprintf(w, "PASS")
	} else {
		fmt.Fprintln(w, "FAIL")
	}
}

// Func is a test function that is expected to fail if invoked.
func Func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "FAIL")
}
