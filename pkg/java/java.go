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

// Package java contains Java buildpack library code.
package java

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// re matches lines in the manifest for a Main-Class entry to detect which jar is appropriate for execution.
	re = regexp.MustCompile("^Main-Class: [^\n]+")
)

// ExecutableJar looks for the jar with a Main-Class manifest. If there is not exactly 1 of these jars, throw an error.
func ExecutableJar(ctx *gcp.Context) (string, error) {
	// Maven-built jar(s) in target directory take precedence over existing jars at app root.
	jars := ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "target/*.jar"))
	if len(jars) == 0 {
		jars = ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "*.jar"))
	}

	// There may be multiple jars due to some frameworks like Quarkus creating multiple jars,
	// so we look for the jar that contains a Main-Class entry in its manifest.
	var executables []string
	for _, jar := range jars {
		if hasMain, err := hasMainManifestEntry(jar); err != nil {
			return "", fmt.Errorf("finding Main-Class manifest: %w", err)
		} else if hasMain {
			executables = append(executables, jar)
		}
	}
	if len(executables) == 0 {
		return "", gcp.UserErrorf("did not find any jar files with a Main-Class manifest entry")
	}
	if len(executables) > 1 {
		return "", gcp.UserErrorf("found more than one jar with a Main-Class manifest entry: %v, please specify an entrypoint", executables)
	}
	return executables[0], nil
}

func hasMainManifestEntry(jar string) (bool, error) {
	r, err := zip.OpenReader(jar)
	if err != nil {
		return false, gcp.UserErrorf("unzipping jar %s: %v", jar, err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != "META-INF/MANIFEST.MF" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false, fmt.Errorf("opening file %s in jar %s: %v", f.FileInfo().Name(), jar, err)
		}
		return hasMain(rc), nil
	}
	return false, nil
}

func hasMain(r io.Reader) bool {
	s := bufio.NewScanner(r)
	for s.Scan() {
		if re.MatchString(s.Text()) {
			return true
		}
	}
	return false
}
