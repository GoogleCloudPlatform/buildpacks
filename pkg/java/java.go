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
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
)

const (
	dateFormat = time.RFC3339Nano
	// repoExpiration is an arbitrary amount of time of 10 weeks to refresh the cache layer.
	// TODO(b/148099877): Investigate proper cache-clearing strategy.
	repoExpiration = time.Duration(time.Hour * 24 * 7 * 10)
	// ManifestPath specifies the path of MANIFEST.MF relative to the working directory.
	ManifestPath = "META-INF/MANIFEST.MF"

	expiryTimestampKey = "expiry_timestamp"
)

var (
	// re matches lines in the manifest for a Main-Class entry to detect which jar is appropriate for execution. For some reason, it does not like `(?m)^Main-Class: [^\s]+`.
	re = regexp.MustCompile("(?m)^Main-Class: [^\r\n\t\f\v ]+")
)

// ExecutableJar looks for the jar with a Main-Class manifest. If there is not exactly 1 of these jars, throw an error.
func ExecutableJar(ctx *gcp.Context) (string, error) {
	// Maven-built jar(s) in target directory take precedence over existing jars at app root.
	jars := ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "target", "*.jar"))
	if len(jars) == 0 {
		jars = ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "build", "libs", "*.jar"))
	}
	if len(jars) == 0 {
		jars = ctx.Glob(filepath.Join(ctx.ApplicationRoot(), "*.jar"))
	}

	// There may be multiple jars due to some frameworks like Quarkus creating multiple jars,
	// so we look for the jar that contains a Main-Class entry in its manifest.
	var executables []string
	for _, jar := range jars {
		if hasMain, err := hasMainManifestEntry(jar); err != nil {
			ctx.Warnf("Failed to inspect %s, skipping: %v.", jar, err)
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
		if f.Name != ManifestPath {
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
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return false
	}
	return re.Match(content)
}

// MainFromManifest returns the main class specified in the manifest at the input path.
func MainFromManifest(ctx *gcp.Context, manifestPath string) (string, error) {
	content := ctx.ReadFile(manifestPath)
	match := re.Find(content)
	if len(match) != 0 {
		return strings.TrimPrefix(string(match), "Main-Class: "), nil
	}
	return "", gcp.UserErrorf("no Main-Class manifest entry found in %s", manifestPath)
}

// CheckCacheExpiration clears the m2 layer and sets a new expiry timestamp when the cache is past expiration.
func CheckCacheExpiration(ctx *gcp.Context, m2CachedRepo *libcnb.Layer) {
	t := time.Now()
	expiry := ctx.GetMetadata(m2CachedRepo, expiryTimestampKey)
	if expiry != "" {
		var err error
		t, err = time.Parse(dateFormat, expiry)
		if err != nil {
			ctx.Debugf("Could not parse expiration date %q, assuming now: %v", expiry, err)
		}
	}
	if t.After(time.Now()) {
		return
	}

	ctx.Debugf("Cache expired on %v, clearing", t)
	ctx.ClearLayer(m2CachedRepo)
	ctx.SetMetadata(m2CachedRepo, expiryTimestampKey, time.Now().Add(repoExpiration).Format(dateFormat))
}
