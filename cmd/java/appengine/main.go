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

// Implements /bin/build for java/appengine buildpack.
package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/appengine"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// re matches lines in the manifest for a Main-Class entry to detect which jar is appropriate for execution.
	re = regexp.MustCompile("^Main-Class: [^\n]+")
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	if os.Getenv(env.Entrypoint) != "" {
		return nil
	}
	if ctx.HasAtLeastOne(ctx.ApplicationRoot(), "*.jar") {
		return nil
	}
	if ctx.FileExists("pom.xml") {
		return nil
	}
	ctx.OptOut("no entrypoint specified, expected at least 1 jar or pom.xml, found neither")
	return nil // Unreachable due to OptOut.
}

func generateEntrypoint(ctx *gcp.Context) (*appengine.Entrypoint, error) {
	if ctx.FileExists("WEB-INF", "appengine-web.xml") {
		return nil, gcp.UserErrorf("appengine-web.xml found, GAE Java compat apps are not supported on Java 11")
	}

	// Maven-built jar(s) in target directory take precedence over existing jars at app root.
	jars := ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "target/*.jar"))
	if len(jars) == 0 {
		jars = ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "*.jar"))
	}

	// There may be multiple jars due to some frameworks like Quarkus creating multiple jars,
	// so we look for the jar that contains a Main-Class entry in its manifest.
	executable, err := executableJar(jars)
	if err != nil {
		return nil, fmt.Errorf("finding executable jar: %w", err)
	}

	return &appengine.Entrypoint{
		Type:    appengine.EntrypointGenerated.String(),
		Command: "/serve " + executable,
	}, nil
}

func buildFn(ctx *gcp.Context) error {
	return appengine.Build(ctx, "java", generateEntrypoint)
}

// executableJar looks for the jar with a Main-Class manifest. If there is not exactly 1 of these jars, throw an error.
func executableJar(jars []string) (string, error) {
	executable := ""
	for _, jar := range jars {
		var hasMain bool
		var err error
		if hasMain, err = hasMainManifestEntry(jar); err != nil {
			return "", fmt.Errorf("finding Main-Class manifest: %w", err)
		}
		if hasMain {
			if executable != "" {
				return "", gcp.UserErrorf("found more than 1 jar with a Main-Class manifest entry")
			}
			executable = jar
		}
	}
	if executable == "" {
		return "", gcp.UserErrorf("could not find a jar with a Main-Class manifest entry")
	}
	return executable, nil
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
