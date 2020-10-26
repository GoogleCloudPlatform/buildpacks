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

// Package main serves to import the functions framework so that it can be fetched via `go mod vendor`.
// A package must be used in code in order for `go mod vendor` to download it.
package main

import (
	_ "github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

func main() {}
