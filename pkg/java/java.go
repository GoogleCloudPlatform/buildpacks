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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb/v2"
)

const (
	dateFormat = time.RFC3339Nano
	// repoExpiration is an arbitrary amount of time of 10 weeks to refresh the cache layer.
	// TODO(b/148099877): Investigate proper cache-clearing strategy.
	repoExpiration = time.Duration(time.Hour * 24 * 7 * 10)
	// ManifestPath specifies the path of MANIFEST.MF relative to the working directory.
	ManifestPath = "META-INF/MANIFEST.MF"
	mainClassKey = "Main-Class"
	// manifestRegexTemplate is a regexp template that matches lines in the manifest for a given entry.
	manifestRegexTemplate = `(?m)^%s: \S+`
	expiryTimestampKey    = "expiry_timestamp"

	// FFJarPathEnv is an environment variable which is used to store the path to the functions framework invoker jar.
	FFJarPathEnv = "GOOGLE_INTERNAL_FUNCTIONS_FRAMEWORK_JAR"

	// GradleBuildArgs is an env var used to append arguments to the gradle build command.
	// Example: `clean assemble` for Maven apps run "gradle clean assemble" command.
	GradleBuildArgs = "GOOGLE_GRADLE_BUILD_ARGS"

	// MavenBuildArgs is an env var used to append arguments to the mvn build command.
	// Example: `clean package` for Maven apps run "mvn clean package" command.
	MavenBuildArgs = "GOOGLE_MAVEN_BUILD_ARGS"
)

var (
	// jarPaths contains the paths that we search for executable jar files. Order of paths decides precedence.
	jarPaths = [][]string{
		[]string{"target"},
		[]string{"build"},
		[]string{"build", "libs"},
		[]string{"*", "build", "libs"},
		// An empty file path searches the application root for jars.
		[]string{},
	}
)

// ExecutableJar looks for the jar with a Main-Class manifest. If there is not exactly 1 of these jars, throw an error.
func ExecutableJar(ctx *gcp.Context) (string, error) {
	var buildable = os.Getenv(env.Buildable)
	if buildable != "" {
		jarPaths = append([][]string{[]string{buildable, "target"}}, jarPaths...)
	}
	for i, path := range jarPaths {
		path = append([]string{ctx.ApplicationRoot()}, path...)
		path = append(path, "*.jar")
		jars, err := ctx.Glob(filepath.Join(path...))
		if err != nil {
			return "", fmt.Errorf("finding jars: %w", err)
		}
		// There may be multiple jars due to some frameworks like Quarkus creating multiple jars,
		// so we look for the jar that contains a Main-Class entry in its manifest.
		executables := filterExecutables(ctx, jars)
		// We've found a path with exactly 1 jar, so return that jar.
		if len(executables) == 1 {
			return executables[0], nil
		} else if len(executables) > 1 {
			return "", gcp.UserErrorf("found more than one jar with a Main-Class manifest entry in %s: %v, please specify an entrypoint", jarPaths[i], executables)
		}
	}
	return "", gcp.UserErrorf("did not find any jar files with a Main-Class manifest entry")
}

func filterExecutables(ctx *gcp.Context, jars []string) []string {
	var executables []string
	for _, jar := range jars {
		if main, err := FindManifestValueFromJar(jar, mainClassKey); err != nil {
			ctx.Warnf("Failed to inspect %s, skipping: %v.", jar, err)
		} else if main != "" {
			executables = append(executables, jar)
		}
	}
	return executables
}

// MainManifestEntry returns the Main-Class manifest entry of the jar at the given filepath,
// or an empty string if the entry does not exist.
func MainManifestEntry(jar string) (string, error) {
	return FindManifestValueFromJar(jar, mainClassKey)
}

// FindManifestValueFromJar returns a manifest entry value from a JAR if found, or empty otherwise.
func FindManifestValueFromJar(jarPath, key string) (string, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", gcp.UserErrorf("unzipping jar %s: %v", jarPath, err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != ManifestPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("opening file %s in jar %s: %v", f.FileInfo().Name(), jarPath, err)
		}
		content, err := ioutil.ReadAll(rc)
		if err != nil {
			return "", err
		}
		return findValueFromManifest(content, key)
	}
	return "", nil
}

// MainFromManifest returns the main class specified in the manifest at the input path.
func MainFromManifest(ctx *gcp.Context, manifestPath string) (string, error) {
	content, err := ctx.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	main, err := findValueFromManifest(content, mainClassKey)
	if err != nil {
		return "", err
	}
	if main == "" {
		return "", gcp.UserErrorf("no Main-Class manifest entry found in the manifest:\n%s", content)
	}
	return main, nil
}

func findValueFromManifest(manifestContent []byte, key string) (string, error) {
	reRaw := fmt.Sprintf(manifestRegexTemplate, key)
	re, err := regexp.Compile(reRaw)
	if err != nil {
		return "", fmt.Errorf("invalid manifest key unsuitable for regexp: %q, %w", key, err)
	}
	match := re.Find(manifestContent)
	if len(match) != 0 {
		return strings.TrimPrefix(string(match), key+": "), nil
	}
	return "", nil
}

// CheckCacheExpiration clears the m2 layer and sets a new expiry timestamp when the cache is past expiration.
func CheckCacheExpiration(ctx *gcp.Context, m2CachedRepo *libcnb.Layer) error {
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
		return nil
	}

	ctx.Debugf("Cache expired on %v, clearing", t)
	if err := ctx.ClearLayer(m2CachedRepo); err != nil {
		return fmt.Errorf("clearing layer %q: %w", m2CachedRepo.Name, err)
	}
	ctx.SetMetadata(m2CachedRepo, expiryTimestampKey, time.Now().Add(repoExpiration).Format(dateFormat))
	return nil
}

// MvnCmd returns the command that should be used to invoke maven for this build.
func MvnCmd(ctx *gcp.Context) (string, error) {
	exists, err := ctx.FileExists("mvnw")
	if err != nil {
		return "", err
	}
	// If this project has the Maven Wrapper, we should use it
	if exists {
		return "./mvnw", nil
	}
	return "mvn", nil
}

// GradleCmd returns the command that should be used to invoke gradle for this build.
func GradleCmd(ctx *gcp.Context) (string, error) {
	exists, err := ctx.FileExists("gradlew")
	if err != nil {
		return "", err
	}
	// If this project has the Gradle Wrapper, we should use it
	if exists {
		return "./gradlew", nil
	}
	return "gradle", nil
}
