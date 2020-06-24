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

// Package acceptance implements functions for builder acceptance tests.
//
// These tests only run locally and require the following command-line tools:
// * pack: https://buildpacks.io/docs/install-pack/
// * container-structure-test: https://github.com/GoogleContainerTools/container-structure-test#installation
package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	// cacheHitMessage is emitted by ctx.CacheHit(). Must match gcpbuildpack value.
	cacheHitMessage = "***** CACHE HIT:"

	// cacheMissMessage is emitted by ctx.CacheMiss(). Must match gcpbuildpack value.
	cacheMissMessage = "***** CACHE MISS:"
)

var (
	testData            string // Path to directory or archive containing source test data.
	structureTestConfig string // Path to container test configuration file.
	builderSource       string // Path to directory or archive containing builder source.
	builderImage        string // Name of the builder image to test; takes precendence over builderSource.
	builderPrefix       string // Prefix for created builder image.
	keepArtifacts       bool   // If true, keeps intermediate artifacts such as application images.
	packBin             string // Path to pack binary.
	structureBin        string // Path to container-structure-test binary.
	lifecycle           string // Path to lifecycle archive; optional.
	pullImages          bool   // Pull stack images instead of using local daemon.
	cloudbuild          bool   // Use cloudbuild network; required for Cloud Build.

	specialChars = regexp.MustCompile("[^a-zA-Z0-9]+")
)

func init() {
	// HOME may be unset with Bazel (https://github.com/bazelbuild/bazel/issues/10652)
	// but `pack` requires that it should be set.
	if _, found := os.LookupEnv("HOME"); !found {
		// The Test Encyclopedia says HOME shouldbe $TEST_TMPDIR
		os.Setenv("HOME", os.Getenv("TEST_TMPDIR"))
	}
}

// DefineFlags sets up flags that control the behavior of the test runner.
func DefineFlags() {
	flag.StringVar(&testData, "test-data", "", "Location of the test data files.")
	flag.StringVar(&structureTestConfig, "structure-test-config", "", "Location of the container structure test configuration.")
	flag.StringVar(&builderSource, "builder-source", "", "Location of the builder source files.")
	flag.StringVar(&builderImage, "builder-image", "", "Name of the builder image to test; takes precedence over builderSource.")
	flag.StringVar(&builderPrefix, "builder-prefix", "acceptance-test-builder-", "Prefix for the generated builder image.")
	flag.BoolVar(&keepArtifacts, "keep-artifacts", false, "Keep images and other artifacts after tests have finished.")
	flag.StringVar(&packBin, "pack", "pack", "Path to pack binary.")
	flag.StringVar(&structureBin, "structure-test", "container-structure-test", "Path to container-structure-test.")
	flag.StringVar(&lifecycle, "lifecycle", "", "Location of lifecycle archive. Overrides builder.toml if specified.")
	flag.BoolVar(&pullImages, "pull-images", true, "Pull stack images before running the tests.")
	flag.BoolVar(&cloudbuild, "cloudbuild", false, "Use cloudbuild network; required for Cloud Build.")
}

// UnarchiveTestData extracts the test-data tgz into a temp dir and returns a cleanup function to be deferred.
// This function overwrites the "test-data" to the /tmp/test-data-* directory that is created.
// Call this function from TestMain if passing test-data as an archive instead of a directory.
// Don't forget to call flag.Parse() first.
func UnarchiveTestData() func() {
	tmpDir, err := ioutil.TempDir("", "test-data-")
	if err != nil {
		log.Fatalf("Creating temp directory: %v", err)
	}
	if _, err = runOutput("tar", "xzCf", tmpDir, testData); err != nil {
		log.Fatalf("Extracting test data archive: %v", err)
	}
	testData = tmpDir
	return func() {
		if keepArtifacts {
			return
		}
		if err = os.RemoveAll(tmpDir); err != nil {
			log.Printf("removing temp directory for test-data: %v", err)
		}
	}
}

// Test describes an acceptance test.
type Test struct {
	// Name specifies the name of the application, if not provided App will be used.
	Name string
	// App specifies the path to the application in testdata.
	App string
	// Path specifies the URL path to send HTTP requests to.
	Path string
	// Env specifies build environment variables as KEY=VALUE strings.
	Env []string
	// RunEnv specifies run environment variables as KEY=VALUE strings.
	RunEnv []string
	// SkipCacheTest skips testing of cached builds for this test case.
	SkipCacheTest bool
	// MustUse specifies the IDs of the buildpacks that must be used during the build.
	MustUse []string
	// MustNotUse specifies the IDs of the buildpacks that must not be used during the build.
	MustNotUse []string
	// FilesMustExist specifies names of files that must exist in the final image.
	FilesMustExist []string
	// FilesMustNotExist specifies names of files that must not exist in the final image.
	FilesMustNotExist []string
	// MustOutput specifies strings to be found in the build logs.
	MustOutput []string
	// MustNotOutput specifies strings to not be found in the build logs.
	MustNotOutput []string
	// MustOutputCached specifies strings to be found in the build logs of a cached build.
	MustOutputCached []string
	// MustNotOutputCached specifies strings to not be found in the build logs of a cached build.
	MustNotOutputCached []string
	// MustRebuildOnChange specifies a file that, when changed in Dev Mode, triggers a rebuild.
	MustRebuildOnChange string
}

// TestApp builds and a single application and verifies that it runs and handles requests.
func TestApp(t *testing.T, builder string, cfg Test) {
	t.Helper()

	env := envSliceAsMap(t, cfg.Env)
	env["GOOGLE_DEBUG"] = "true"

	if cfg.Name == "" {
		cfg.Name = cfg.App
	}
	// Docker image names may not contain underscores or start with a capital letter.
	image := fmt.Sprintf("%s-%s", strings.ToLower(specialChars.ReplaceAllString(cfg.Name, "-")), builder)

	// Delete the docker image and volumes created by pack during the build.
	defer func() {
		cleanUpVolumes(t, image)
		cleanUpImage(t, image)
	}()

	// Create a configuration for container-structure-tests.
	checks := NewStructureTest(cfg.FilesMustExist, cfg.FilesMustNotExist)

	// Run a no-cache build, followed by a cache build, unless caching is disabled for the app.
	cacheOptions := []bool{false}
	if !cfg.SkipCacheTest {
		cacheOptions = append(cacheOptions, true)
	}
	for _, cache := range cacheOptions {
		t.Run(fmt.Sprintf("cache %t", cache), func(t *testing.T) {
			buildApp(t, cfg.App, image, builder, env, cache, cfg)
			verifyBuildMetadata(t, image, cfg.MustUse, cfg.MustNotUse)
			verifyStructure(t, cfg.App, image, builder, cache, checks)
			invokeApp(t, cfg, image, cache)
		})
	}
}

// FailureTest describes a failure test.
type FailureTest struct {
	// Name specifies the name of the application, if not provided App will be used.
	Name string
	// App specifies the path to the application in testdata.
	App string
	// Env specifies build environment variables as KEY=VALUE strings.
	Env []string
	// MustMatch specifies a string that must appear in the builder output.
	MustMatch string
	// SkipBuilderOutputMatch is true if the MustMatch string is not expected in $BUILDER_OUTPUT.
	SkipBuilderOutputMatch bool
}

// TestBuildFailure runs a build and ensures that it fails. Additionally, it ensures the emitted logs match mustMatch regexps.
func TestBuildFailure(t *testing.T, builder string, cfg FailureTest) {
	t.Helper()

	env := envSliceAsMap(t, cfg.Env)
	env["GOOGLE_DEBUG"] = "true"
	if !cfg.SkipBuilderOutputMatch {
		env["BUILDER_OUTPUT"] = "/tmp/builderoutput"
		env["EXPECTED_BUILDER_OUTPUT"] = cfg.MustMatch
	}

	if cfg.Name == "" {
		cfg.Name = cfg.App
	}
	image := fmt.Sprintf("%s-%s", strings.ToLower(specialChars.ReplaceAllString(cfg.Name, "-")), builder)

	// Delete the docker volumes created by pack during the build.
	if !keepArtifacts {
		defer cleanUpVolumes(t, image)
	}

	outb, errb, cleanup := buildFailingApp(t, cfg.App, image, builder, env)
	defer cleanup()

	r, err := regexp.Compile(cfg.MustMatch)
	if err != nil {
		t.Fatalf("regexp %q failed to compile: %v", r, err)
	}
	if r.Match(outb) {
		t.Logf("Expected regexp %q found in stdout.", r)
	} else if r.Match(errb) {
		t.Logf("Expected regexp %q found in stderr.", r)
	} else {
		t.Errorf("Expected regexp %q not found in stdout or stderr:\n\nstdout:\n\n%s\n\nstderr:\n\n%s", r, outb, errb)
	}
	expectedLog := "Expected pattern included in error output: true"
	if !cfg.SkipBuilderOutputMatch && !strings.Contains(string(errb), expectedLog) {
		t.Errorf("Expected regexp %q not found in BUILDER_OUTPUT", r)
	}
}

// invokeApp performs an HTTP GET on the app.
func invokeApp(t *testing.T, cfg Test, image string, cache bool) {
	t.Helper()

	containerID, host, port, cleanup := startContainer(t, image, cfg.RunEnv, cache)
	defer cleanup()

	// Check that the application responds with `PASS`.
	start := time.Now()

	body, status, statusCode, err := invokeHTTP(host, port, cfg.Path)
	if err != nil {
		t.Fatalf("Unable to invoke app: %v", err)
	}

	t.Logf("Got response: status %v, body %q (in %s)", status, body, time.Since(start))

	if want := http.StatusOK; statusCode != want {
		t.Errorf("Unexpected status code: got %d, want %d", statusCode, want)
	}
	if want := "PASS"; !strings.HasSuffix(body, want) {
		t.Errorf("Response body does not contain suffix: got %q, want %q", body, want)
	}

	if cfg.MustRebuildOnChange != "" {
		// Modify a source file in the running container.
		if _, err := runOutput("docker", "exec", containerID, "sed", "-i", "s/PASS/UPDATED/", cfg.MustRebuildOnChange); err != nil {
			t.Fatalf("Unable to modify a source file in the running container %q: %v", containerID, err)
		}

		// Check that the application responds with `UPDATED`.
		for try := 10; try >= 1; try-- {
			time.Sleep(1 * time.Second)

			body, _, _, err := invokeHTTP(host, port, cfg.Path)
			if err != nil {
				continue
			}

			want := "UPDATED"
			if body == want {
				break
			}
			if try == 1 {
				t.Errorf("Wrong body: got %q, want %q", body, want)
			}
		}
	}
}

// invokeHTTP makes an http call to a given host:port/path. Returns the body, status
// and statusCode of the response.
func invokeHTTP(host string, port int, path string) (string, string, int, error) {
	var res *http.Response
	var err error

	// Try to connect the the container until it succeeds (up to 60s).
	for retry := 0; retry < 600; retry++ {
		res, err = http.Get(fmt.Sprintf("http://%s:%d%s", host, port, path))
		if err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// The connection never succeeded.
	if err != nil {
		return "", "", 0, fmt.Errorf("error making request: %w", err)
	}

	// The connection was a success.
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("error reading body: %w", err)
	}
	res.Body.Close()

	return strings.TrimSpace(string(bytes)), res.Status, res.StatusCode, nil
}

// randString generates a random string of length n.
func randString(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}

// runOutput runs the given command and returns its stdout or an error.
func runOutput(args ...string) (string, error) {
	log.Printf("Running %v\n", args)
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		logs := ""
		if ee, ok := err.(*exec.ExitError); ok {
			logs = fmt.Sprintf("\nstdout:\n%s\nstderr:\n%s\n", out, ee.Stderr)
		}
		return "", fmt.Errorf("running command %v: %v%s", args, err, logs)
	}
	return strings.TrimSpace(string(out)), nil
}

// cleanUpImage attempts to delete an image from the Docker daemon.
func cleanUpImage(t *testing.T, name string) {
	t.Helper()

	if keepArtifacts {
		return
	}
	if _, err := runOutput("docker", "rmi", "-f", name); err != nil {
		t.Logf("Failed to clean up image: %v", err)
	}
}

// CreateBuilder creates a builder image.
func CreateBuilder(t *testing.T) (string, func()) {
	t.Helper()

	name := builderPrefix + randString(10)

	if builderImage != "" {
		t.Logf("Testing existing builder image: %s", builderImage)
		if pullImages {
			if _, err := runOutput("docker", "pull", builderImage); err != nil {
				t.Fatalf("Error pulling %s: %v", builderImage, err)
			}
			run, err := runImageFromMetadata(builderImage)
			if err != nil {
				t.Fatalf("Error extracting run image from image %s: %v", builderImage, err)
			}
			if _, err := runOutput("docker", "pull", run); err != nil {
				t.Fatalf("Error pulling %s: %v", run, err)
			}
		}
		// Pack cache is based on builder name; retag with a unique name.
		if _, err := runOutput("docker", "tag", builderImage, name); err != nil {
			t.Fatalf("Error tagging %s as %s: %v", builderImage, name, err)
		}
		return name, func() {
			cleanUpImage(t, name)
		}
	}

	builderLoc, cleanUpBuilder := extractBuilder(t, builderSource)
	config := filepath.Join(builderLoc, "builder.toml")

	if lifecycle != "" {
		t.Logf("Using lifecycle location: %s", lifecycle)
		if c, err := updateLifecycle(config, lifecycle); err != nil {
			t.Fatalf("Error updating lifecycle location: %v", err)
		} else {
			config = c
		}
	}

	run, build, err := stackImagesFromConfig(config)
	if err != nil {
		t.Fatalf("Error extracting stack images from %s: %v", config, err)
	}
	// Pull images once in the beginning to prevent them from changing in the middle of testing.
	// The images are intentionally not cleaned up to prevent conflicts across different test targets.
	if pullImages {
		if _, err := runOutput("docker", "pull", run); err != nil {
			t.Fatalf("Error pulling %s: %v", run, err)
		}
		if _, err := runOutput("docker", "pull", build); err != nil {
			t.Fatalf("Error pulling %s: %v", build, err)
		}
	}

	// Pack command to create the builder.
	args := strings.Fields(fmt.Sprintf("create-builder %s --builder-config %s --no-pull --verbose --no-color", name, config))
	cmd := exec.Command(packBin, args...)

	outFile, errFile, cleanup := outFiles(t, name, "pack", "create-builder")
	defer cleanup()
	var errb bytes.Buffer
	cmd.Stdout = outFile
	cmd.Stderr = io.MultiWriter(errFile, &errb) // pack emits buildpack output to stderr.

	start := time.Now()
	t.Logf("Creating builder (logs %s)", filepath.Dir(outFile.Name()))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error creating builder: %v (%v)", err, errb.String())
	}
	t.Logf("Successfully created builder: %s (in %s)", name, time.Since(start))

	return name, func() {
		cleanUpImage(t, name)
		cleanUpBuilder()
	}
}

func extractBuilder(t *testing.T, builderSource string) (string, func()) {
	t.Helper()

	if !strings.HasSuffix(builderSource, ".tar") {
		return builderSource, func() {}
	}

	start := time.Now()
	d, err := ioutil.TempDir("", "builder-")
	if err != nil {
		t.Fatalf("Error creating temp builder location: %v", err)
	}
	out, err := runOutput("tar", "xCf", d, builderSource)
	if err != nil {
		t.Fatalf("Error extracting %s: %s\n%s\n", builderSource, err, out)
	}
	t.Logf("Successfully extracted builder to %s (in %s)", d, time.Since(start))

	return d, func() {
		if !keepArtifacts {
			os.RemoveAll(d)
		}
	}
}

// updateLifecycle rewrites the lifecycle field of the config to the given uri.
func updateLifecycle(config, uri string) (string, error) {
	p, err := ioutil.ReadFile(config)
	if err != nil {
		return "", fmt.Errorf("reading %s: %v", config, err)
	}
	var data map[string]interface{}
	if err := toml.Unmarshal(p, &data); err != nil {
		return "", fmt.Errorf("unmarshaling %s: %v", config, err)
	}

	data["lifecycle"] = map[string]string{
		"uri": uri,
	}

	f, err := ioutil.TempFile("", "builder-*.toml")
	if err != nil {
		return "", fmt.Errorf("creating temporary file: %v", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(data); err != nil {
		return "", fmt.Errorf("writing data: %v", err)
	}
	// Buildpack paths are relative; move to original directory.
	orig := f.Name()
	dest := filepath.Join(filepath.Dir(config), filepath.Base(orig))
	if err := os.Rename(orig, dest); err != nil {
		return "", fmt.Errorf("renaming %s to %s: %v", orig, dest, err)
	}
	return dest, nil
}

// runImageFromMetadata returns the run image name from the metadata of the given image.
func runImageFromMetadata(image string) (string, error) {
	format := "--format={{(index (index .Config.Labels) \"io.buildpacks.builder.metadata\")}}"
	out, err := runOutput("docker", "inspect", image, format)
	if err != nil {
		return "", fmt.Errorf("reading builer metadata: %v", err)
	}

	var metadata struct {
		Stack struct {
			RunImage struct {
				Image string `json:"image"`
			} `json:"runImage"`
		} `json:"stack"`
	}

	if err := json.Unmarshal([]byte(out), &metadata); err != nil {
		return "", fmt.Errorf("error unmarshalling build metadata: %v", err)
	}
	return metadata.Stack.RunImage.Image, nil
}

// stackImagesFromConfig returns the run images specified by the given builder.toml.
func stackImagesFromConfig(path string) (string, string, error) {
	p, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("reading %s: %v", path, err)
	}
	var config struct {
		Stack struct {
			RunImage   string `toml:"run-image"`
			BuildImage string `toml:"build-image"`
		} `toml:"stack"`
	}
	if err := toml.Unmarshal(p, &config); err != nil {
		return "", "", fmt.Errorf("unmarshaling %s: %v", path, err)
	}
	return config.Stack.RunImage, config.Stack.BuildImage, nil
}

func buildCommand(app, image, builder string, env map[string]string, cache bool) []string {
	src := filepath.Join(testData, app)
	// Pack command to build app.
	args := strings.Fields(fmt.Sprintf("%s build %s --builder %s --path %s --no-pull --verbose --no-color --trust-builder", packBin, image, builder, src))
	if !cache {
		args = append(args, "--clear-cache")
	}
	for k, v := range env {
		args = append(args, "--env", fmt.Sprintf("%s=%s", k, v))
	}
	// Prevents a race condition in pack when concurrently running builds with the same builder.
	// Pack generates an "emphemeral builder" that contains env vars, adding an env var with a random
	// value ensures that the generated builder sha is unique and removing it after one build will
	// not affect other builds running concurrently.
	args = append(args, "--env", "GOOGLE_RANDOM="+randString(8))
	log.Printf("Running %v\n", args)
	return args
}

// buildApp builds an application image from source.
func buildApp(t *testing.T, app, image, builder string, env map[string]string, cache bool, cfg Test) {
	t.Helper()

	bcmd := buildCommand(app, image, builder, env, cache)
	cmd := exec.Command(bcmd[0], bcmd[1:]...)

	outFile, errFile, cleanup := outFiles(t, builder, "pack-build", fmt.Sprintf("%s-cache-%t", image, cache))
	defer cleanup()

	var errb bytes.Buffer
	cmd.Stdout = outFile
	cmd.Stderr = io.MultiWriter(errFile, &errb) // pack emits buildpack output to stderr.

	start := time.Now()
	t.Logf("Building application (logs %s)", filepath.Dir(outFile.Name()))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error building application: %v (%v)", err, errb.String())
	}

	// Check that expected output is found in the logs.
	mustOutput := cfg.MustOutput
	mustNotOutput := cfg.MustNotOutput
	if cache {
		mustOutput = cfg.MustOutputCached
		mustNotOutput = cfg.MustNotOutputCached
	}

	for _, text := range mustOutput {
		if !strings.Contains(errb.String(), text) {
			t.Errorf("Build logs must contain %q:\n%s", text, errb.String())
		}
	}
	for _, text := range mustNotOutput {
		if strings.Contains(errb.String(), text) {
			t.Errorf("Build logs must not contain %q:\n%s", text, errb.String())
		}
	}

	// Scan for incorrect cache hits/misses.
	if cache {
		if strings.Contains(errb.String(), cacheMissMessage) {
			t.Fatalf("FAIL: Cached build had a cache miss:\n%s", errb.String())
		}
	} else {
		if strings.Contains(errb.String(), cacheHitMessage) {
			t.Fatalf("FAIL: Non-cache build had a cache hit:\n%s", errb.String())
		}
	}

	t.Logf("Successfully built application: %s (in %s)", image, time.Since(start))
}

// buildFailingApp attempts to build an app and ensures that it failues (non-zero exit code).
// It returns the build's stdout, stderr and a cleanup function.
func buildFailingApp(t *testing.T, app, image, builder string, env map[string]string) ([]byte, []byte, func()) {
	t.Helper()

	bcmd := buildCommand(app, image, builder, env, false)
	cmd := exec.Command(bcmd[0], bcmd[1:]...)

	outFile, errFile, cleanup := outFiles(t, builder, "pack-build-failing", image)
	defer cleanup()
	var outb, errb bytes.Buffer
	cmd.Stdout = io.MultiWriter(outFile, &outb)
	cmd.Stderr = io.MultiWriter(errFile, &errb)

	t.Logf("Building application expected to fail (logs %s)", filepath.Dir(outFile.Name()))
	if err := cmd.Run(); err == nil {
		// No error, but we expected one; this is a test failure.
		t.Fatal("Application built successfully, but should not have.")
	} else {
		// We got an error, but we need to check that it's due to a non-zero exit code (which is what
		// we want in this case). If the error is an ExitError, it was a non-zero exit code.
		// Otherwise, it a truly unexpected error.
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("executing command %q: %v", bcmd, err)
		} else {
			t.Logf("Application build failed as expected: %s", image)
		}
	}

	return outb.Bytes(), errb.Bytes(), func() {
		cleanUpImage(t, image)
	}
}

// verifyStructure verifies the structure of the image.
func verifyStructure(t *testing.T, app, image, builder string, cache bool, checks *StructureTest) {
	t.Helper()

	start := time.Now()
	configurations := []string{structureTestConfig}

	if checks != nil {
		tmpDir, err := ioutil.TempDir("", "container-structure-tests")
		if err != nil {
			t.Fatalf("Error creating temp directory: %v", err)
		}
		defer func() {
			if keepArtifacts {
				return
			}
			if err = os.RemoveAll(tmpDir); err != nil {
				log.Printf("Removing temp directory for container-structure-tests: %v", err)
			}
		}()

		// Create a config file for container-structure-tests.
		buf, err := json.Marshal(checks)
		if err != nil {
			t.Fatalf("Marshalling container-structure-tests configuration: %v", err)
		}

		configPath := filepath.Join(tmpDir, "config.json")
		err = ioutil.WriteFile(configPath, buf, 0644)
		if err != nil {
			t.Fatalf("Writing container-structure-tests configuration: %v", err)
		}

		configurations = append(configurations, configPath)
	}

	// Container-structure-test command to test the image.
	args := []string{"test", "--image", image}
	for _, configuration := range configurations {
		args = append(args, "--config", configuration)
	}
	cmd := exec.Command(structureBin, args...)

	outFile, errFile, cleanup := outFiles(t, builder, "container-structure-test", fmt.Sprintf("%s-cache-%t", image, cache))
	defer cleanup()
	var outb bytes.Buffer
	cmd.Stdout = io.MultiWriter(outFile, &outb)
	cmd.Stderr = errFile

	t.Logf("Running structure tests (logs %s)", filepath.Dir(outFile.Name()))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error running structure tests: %v (%v)", err, outb.String())
	}
	t.Logf("Successfully ran structure tests on %s (in %s)", image, time.Since(start))
}

// verifyBuildMetadata verifies the image was built with correct buildpacks.
func verifyBuildMetadata(t *testing.T, image string, mustUse, mustNotUse []string) {
	t.Helper()

	start := time.Now()
	out, err := runOutput("docker", "inspect", "--format={{index .Config.Labels \"io.buildpacks.build.metadata\"}}", image)
	if err != nil {
		t.Fatalf("Error reading build metadata: %v", err)
	}

	var metadata struct {
		Buildpacks []struct {
			ID string `json:"id"`
		} `json:"buildpacks"`
	}

	if err := json.Unmarshal([]byte(out), &metadata); err != nil {
		t.Errorf("Error unmarshalling build metadata: %v", err)
	}

	usedBuildpacks := map[string]bool{}
	for _, bp := range metadata.Buildpacks {
		usedBuildpacks[bp.ID] = true
	}

	for _, id := range mustUse {
		if _, used := usedBuildpacks[id]; !used {
			t.Fatalf("Must use buildpack %s was not used.", id)
		}
	}

	for _, id := range mustNotUse {
		if _, used := usedBuildpacks[id]; used {
			t.Fatalf("Must not use buildpack %s was used.", id)
		}
	}

	t.Logf("Successfully verified build metadata (in %s)", time.Since(start))
}

// startContainer starts a container for the given app and exposes port 8080.
// The function returns the containerID, the host and port at which the app is reachable and a cleanup function.
func startContainer(t *testing.T, image string, env []string, cache bool) (string, string, int, func()) {
	t.Helper()

	// Start docker container and get its id.
	command := []string{"docker", "run", "--detach", "--publish=8080"}
	for _, e := range env {
		command = append(command, "--env", e)
	}
	if cloudbuild {
		command = append(command, "--network=cloudbuild")
	}
	command = append(command, image)

	id, err := runOutput(command...)
	if err != nil {
		t.Fatalf("Error starting container: %v", err)
	}
	t.Logf("Successfully started container: %s", id)

	// Cloud Build uses the exposed port and assigns an IP address.
	// Local execution uses "localhost" and assigns a random port.
	host, port := "localhost", 8080
	if cloudbuild {
		host = cloudbuildIP(t, id)
	} else {
		port = hostPort(t, id)
	}

	return id, host, port, func() {
		if _, err := runOutput("docker", "stop", id); err != nil {
			t.Logf("Failed to stop container: %v", err)
		}
		if keepArtifacts {
			return
		}
		if _, err := runOutput("docker", "rm", "-f", id); err != nil {
			t.Logf("Failed to clean up container: %v", err)
		}
	}
}

// hostPort returns the host port assigned to the exposed container port.
func hostPort(t *testing.T, id string) int {
	t.Helper()

	format := "--format={{(index (index .NetworkSettings.Ports \"8080/tcp\") 0).HostPort}}"
	portstr, err := runOutput("docker", "inspect", id, format)
	if err != nil {
		t.Fatalf("Error getting port: %v", err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		t.Fatalf("Error converting port to int: %v", err)
	}
	t.Logf("Successfully got port: %d", port)
	return port
}

// cloudbuildIP returns the cloudbuild network IP address of the container.
func cloudbuildIP(t *testing.T, id string) string {
	t.Helper()

	format := "--format={{(index .NetworkSettings.Networks \"cloudbuild\").IPAddress}}"
	ip, err := runOutput("docker", "inspect", id, format)
	if err != nil {
		t.Fatalf("Error getting IP address: %v", err)
	}
	t.Logf("Successfully got IP address: %s", ip)
	return ip
}

func outFiles(t *testing.T, builder, dir, logName string) (outFile, errFile *os.File, cleanup func()) {
	t.Helper()

	tempDir := os.TempDir()
	if blaze := os.Getenv("TEST_UNDECLARED_OUTPUTS_DIR"); blaze != "" {
		tempDir = blaze
	}

	d := filepath.Join(tempDir, "buildpack-acceptance-logs", builder, dir)
	if err := os.MkdirAll(d, 0755); err != nil {
		t.Fatalf("Failed to create logs dir %q: %v", d, err)
	}

	logName = strings.ReplaceAll(logName, "/", "_")
	outName := filepath.Join(d, fmt.Sprintf("%s.stdout", logName))
	errName := filepath.Join(d, fmt.Sprintf("%s.stderr", logName))

	outFile, err := os.Create(outName)
	if err != nil {
		t.Fatalf("Error creating stdout file %q: %v", outName, err)
	}

	errFile, err = os.Create(errName)
	if err != nil {
		t.Fatalf("Error creating stderr file %q: %v", errName, err)
	}

	return outFile, errFile, func() {
		if err := outFile.Close(); err != nil {
			t.Fatalf("failed to close %q: %v", outFile.Name(), err)
		}
		if err := errFile.Close(); err != nil {
			t.Fatalf("failed to close %q: %v", outFile.Name(), err)
		}
	}
}

// cleanUpVolumes tries to delete volumes created by pack during the build.
func cleanUpVolumes(t *testing.T, image string) {
	t.Helper()

	if keepArtifacts {
		return
	}

	// This logic is copied from pack's codebase.
	fqn := "index.docker.io/library/" + image + ":latest"
	digest := sha256.Sum256([]byte(fqn))
	prefix := fmt.Sprintf("pack-cache-%x", digest[:6])

	if _, err := runOutput("docker", "volume", "rm", "-f", prefix+".launch", prefix+".build"); err != nil {
		t.Logf("Failed to clean up cache volumes: %v", err)
	}
}

// envSliceAsMap converts the given slice of KEY=VAL strings to a map.
func envSliceAsMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("Invalid environment variable: %q", kv)
		}
		if _, ok := result[parts[0]]; ok {
			t.Fatalf("Env var %s is already set.", parts[0])
		}
		result[parts[0]] = parts[1]
	}
	return result
}
