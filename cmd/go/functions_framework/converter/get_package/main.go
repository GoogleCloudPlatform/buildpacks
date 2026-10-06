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

// Script to extract Go package from a given source directory.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"strings"
)

var (
	dir = flag.String("dir", "", "Directory containing *.go files from which to extract a package name.")
)

// parsedPackage represents a parsed package.
type parsedPackage struct {
	Name    string              `json:"name"`
	Imports map[string]struct{} `json:"imports"`
}

// extract extracts the name of the package in the specified directory.
// Expects that the specified directory contains one and only one Go package.
func extract(source string) (*parsedPackage, error) {
	fset := token.NewFileSet() // positions are relative to fset

	// Parse all .go files in dir but stop after processing the package.
	pkgs, err := parser.ParseDir(fset, source, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source in %s: %v", source, err)
	}

	// Check if all files belong to the same package
	packageName := ""
	for k := range pkgs {
		if packageName == "" {
			packageName = k
			continue
		}

		if k != packageName {
			return nil, fmt.Errorf("multiple packages in user code directory: %s != %s", packageName, k)
		}
	}

	if packageName == "" {
		return nil, fmt.Errorf("unable to find Go package in %s", source)
	}

	packageImports := map[string]struct{}{}
	for _, fi := range pkgs[packageName].Files {
		for _, im := range fi.Imports {
			packageImports[strings.Trim(im.Path.Value, `"`)] = struct{}{}
		}
	}

	return &parsedPackage{
		Name:    packageName,
		Imports: packageImports,
	}, nil
}

func main() {
	flag.Parse()

	if *dir == "" {
		log.Fatalf("No directory specified.")
	}

	pkg, err := extract(*dir)
	if err != nil {
		log.Fatalf("Unable to extract package name and imports: %v.", err)
	}

	b, err := json.Marshal(pkg)
	if err != nil {
		log.Fatalf("Unable to marshal extracted package name and imports: %v.", err)
	}
	fmt.Print(string(b))
}
