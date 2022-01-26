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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/xid"

	"github.com/GoogleCloudPlatform/buildpacks/internal/checktools"
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
	builderImage        string // Name of the builder image to test; takes precedence over builderSource.
	builderPrefix       string // Prefix for created builder image.
	keepArtifacts       bool   // If true, keeps intermediate artifacts such as application images.
	packBin             string // Path to pack binary.
	structureBin        string // Path to container-structure-test binary.
	lifecycle           string // Path to lifecycle archive; optional.
	pullImages          bool   // Pull stack images instead of using local daemon.
	cloudbuild          bool   // Use cloudbuild network; required for Cloud Build.

	specialChars = regexp.MustCompile("[^a-zA-Z0-9]+")
)

type requestType string

// Different function signature types.
const (
	HTTPType            requestType = "http"
	CloudEventType      requestType = "cloudevent"
	BackgroundEventType requestType = "event"
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
	// Entrypoint specifies the Docker image entrypoint to invoke.
	// All processes are added to PATH so --entrypoint=<process> will start <process>.
	Entrypoint string
	// MustMatch specifies the expected response, if not provided "PASS" will be used.
	MustMatch string
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
	// MustMatchStatusCode specifies the HTTP status code hitting the function endpoint should return.
	MustMatchStatusCode int
	// FlakyBuildAttempts specifies the number of times a failing build should be retried.
	FlakyBuildAttempts int
	// RequestType specifies the payload of the request used to test the function.
	RequestType requestType
	// BOM specifies the list of bill-of-material entries expected in the built image metadata.
	BOM []BOMEntry
	// Setup is a function that sets up the source directory before test.
	Setup setupFunc
}

// setupFunc is a function that is called before the test starts and can be used to modify the test source.
// The function has access to the builder image name and a directory with a modifiable copy of the source
type setupFunc func(builder, srcDir string) error

// BOMEntry represents a bill-of-materials entry in the image metadata.
type BOMEntry struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

	// Run Setup function if provided.
	src := filepath.Join(testData, cfg.App)
	if cfg.Setup != nil {
		src = setupSource(t, cfg.Setup, builder, src, cfg.App)
	}

	// Run a no-cache build, followed by a cache build, unless caching is disabled for the app.
	cacheOptions := []bool{false}
	if !cfg.SkipCacheTest {
		cacheOptions = append(cacheOptions, true)
	}
	for _, cache := range cacheOptions {
		t.Run(fmt.Sprintf("cache %t", cache), func(t *testing.T) {
			buildApp(t, src, image, builder, env, cache, cfg)
			verifyBuildMetadata(t, image, cfg.MustUse, cfg.MustNotUse, cfg.BOM)
			verifyStructure(t, image, builder, cache, checks)
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
	// Setup is a function that sets up the source directory before test.
	Setup setupFunc
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

	src := filepath.Join(testData, cfg.App)
	if cfg.Setup != nil {
		src = setupSource(t, cfg.Setup, builder, src, cfg.App)
	}

	outb, errb, cleanup := buildFailingApp(t, src, image, builder, env)
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

// invokeApp performs an HTTP GET or sends a Cloud Event payload to the app.
func invokeApp(t *testing.T, cfg Test, image string, cache bool) {
	t.Helper()

	containerID, host, port, cleanup := startContainer(t, image, cfg.Entrypoint, cfg.RunEnv, cache)
	defer cleanup()

	// Check that the application responds with `PASS`.
	start := time.Now()

	reqType := HTTPType
	if cfg.RequestType != "" {
		reqType = cfg.RequestType
	}

	body, status, statusCode, err := sendRequest(host, port, cfg.Path, reqType)

	if err != nil {
		t.Fatalf("Unable to invoke app: %v", err)
	}

	t.Logf("Got response: status %v, body %q (in %s)", status, body, time.Since(start))

	wantCode := http.StatusOK
	if cfg.MustMatchStatusCode != 0 {
		wantCode = cfg.MustMatchStatusCode
	}
	if statusCode != wantCode {
		t.Errorf("Unexpected status code: got %d, want %d", statusCode, wantCode)
	}
	if reqType == HTTPType && cfg.MustMatch == "" {
		cfg.MustMatch = "PASS"
	}
	if !strings.HasSuffix(body, cfg.MustMatch) {
		t.Errorf("Response body does not contain suffix: got %q, want %q", body, cfg.MustMatch)
	}

	if cfg.MustRebuildOnChange != "" {
		start = time.Now()
		// Modify a source file in the running container.
		if _, err := runOutput("docker", "exec", containerID, "sed", "-i", "s/PASS/UPDATED/", cfg.MustRebuildOnChange); err != nil {
			t.Fatalf("Unable to modify a source file in the running container %q: %v", containerID, err)
		}

		// Check that the application responds with `UPDATED`.
		tries := 30
		for try := tries; try >= 1; try-- {
			time.Sleep(1 * time.Second)

			body, status, _, err := sendRequestWithTimeout(host, port, cfg.Path, 10*time.Second, reqType)
			// An app that is rebuilding can be unresponsive.
			if err != nil {
				if try == 1 {
					t.Fatalf("Unable to invoke app after updating source with %d attempts: %v", tries, err)
				}
				continue
			}

			want := "UPDATED"
			if body == want {
				t.Logf("Got response: status %v, body %q (in %s)", status, body, time.Since(start))
				break
			}
			if try == 1 {
				t.Errorf("Wrong body: got %q, want %q", body, want)
			}
		}
	}
}

// sendRequest makes an http call to a given host:port/path
// or send a cloud event payload to host:port if sendCloudEvents is true.
// Returns the body, status and statusCode of the response.
func sendRequest(host string, port int, path string, functionType requestType) (string, string, int, error) {
	return sendRequestWithTimeout(host, port, path, 120*time.Second, functionType)
}

// sendRequestWithTimeout makes an http call to a given host:port/path with the specified timeout
// or send a cloud event payload with timeout to host:port if sendCloudEvents is true.
// Returns the body, status and statusCode of the response.
func sendRequestWithTimeout(host string, port int, path string, timeout time.Duration, functionType requestType) (string, string, int, error) {
	var res *http.Response
	var loopErr error

	// Try to connect the the container until it succeeds up to the timeout.
	sleep := 100 * time.Millisecond
	attempts := int(timeout / sleep)
	url := fmt.Sprintf("http://%s:%d%s", host, port, path)
	for attempt := 0; attempt < attempts; attempt++ {
		switch functionType {
		case BackgroundEventType:
			// GCS event example
			beJSON := []byte(`{
				"context": {
				   "eventId": "aaaaaa-1111-bbbb-2222-cccccccccccc",
				   "timestamp": "2020-09-29T11:32:00.000Z",
				   "eventType": "google.storage.object.finalize",
				   "resource": {
					  "service": "storage.googleapis.com",
					  "name": "projects/_/buckets/some-bucket/objects/folder/Test.cs",
					  "type": "storage#object"
				   }
				},
				"data": {
				   "bucket": "some-bucket",
				   "contentType": "text/plain",
				   "crc32c": "rTVTeQ==",
				   "etag": "CNHZkbuF/ugCEAE=",
				   "generation": "1587627537231057",
				   "id": "some-bucket/folder/Test.cs/1587627537231057",
				   "kind": "storage#object",
				   "md5Hash": "kF8MuJ5+CTJxvyhHS1xzRg==",
				   "mediaLink": "https://www.googleapis.com/download/storage/v1/b/some-bucket/o/folder%2FTest.cs?generation=1587627537231057\u0026alt=media",
				   "metageneration": "1",
				   "name": "folder/Test.cs",
				   "selfLink": "https://www.googleapis.com/storage/v1/b/some-bucket/o/folder/Test.cs",
				   "size": "352",
				   "storageClass": "MULTI_REGIONAL",
				   "timeCreated": "2020-04-23T07:38:57.230Z",
				   "timeStorageClassUpdated": "2020-04-23T07:38:57.230Z",
				   "updated": "2020-04-23T07:38:57.230Z"
				}
			  }`)

			res, loopErr = http.Post(url, "application/json", bytes.NewBuffer(beJSON))
		case CloudEventType:
			ceHeaders := map[string]string{
				"Content-Type": "application/cloudevents+json",
			}
			ceJSON := []byte(`{
				"specversion" : "1.0",
				"type" : "com.example.type",
				"source" : "https://github.com/cloudevents/spec/pull",
				"subject" : "123",
				"id" : "A234-1234-1234",
				"time" : "2018-04-05T17:31:00Z",
				"comexampleextension1" : "value",
				"data" : "hello"
			}`)

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(ceJSON))
			if err != nil {
				return "", "", 0, fmt.Errorf("error creating CloudEvent HTTP request: %w", loopErr)
			}

			for k, v := range ceHeaders {
				req.Header.Add(k, v)
			}
			client := &http.Client{}
			res, loopErr = client.Do(req)
		default:
			res, loopErr = http.Get(url)
		}
		if loopErr == nil {
			break
		}

		time.Sleep(sleep)
	}

	// The connection never succeeded.
	if loopErr != nil {
		return "", "", 0, fmt.Errorf("error making request: %w", loopErr)
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
	start := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		logs := ""
		if ee, ok := err.(*exec.ExitError); ok {
			logs = fmt.Sprintf("\nstdout:\n%s\nstderr:\n%s\n", out, ee.Stderr)
		}
		return "", fmt.Errorf("running command %v: %v%s", args, err, logs)
	}
	log.Printf("Finished %v (in %s)\n", args, time.Since(start))
	return strings.TrimSpace(string(out)), nil
}

func runCombinedOutput(args ...string) (string, error) {
	log.Printf("Running %v\n", args)
	start := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logs := fmt.Sprintf("\nstdout & stderr:\n%s\n", out)
		return "", fmt.Errorf("running command %v: %v%s", args, err, logs)
	}
	log.Printf("Finished %v (in %s)\n", args, time.Since(start))
	return string(out), nil
}

// runDockerLogs returns the logs for a container, the lineLimit parameter
// controls the maximum number of lines read from the log
func runDockerLogs(containerID string, lineLimit int) (string, error) {
	return runCombinedOutput("docker", "logs", "--tail", string(lineLimit), containerID)
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

	if err := checktools.Installed(); err != nil {
		t.Fatalf("Error checking tools: %v", err)
	}
	if err := checktools.PackVersion(); err != nil {
		t.Fatalf("Error checking pack version: %v", err)
	}

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
	args := strings.Fields(fmt.Sprintf("builder create %s --config %s --pull-policy never --verbose --no-color", name, config))
	cmd := exec.Command(packBin, args...)

	outFile, errFile, cleanup := outFiles(t, name, "pack", "create-builder")
	defer cleanup()
	var outb, errb bytes.Buffer
	cmd.Stdout = io.MultiWriter(outFile, &outb) // pack emits some errors to stdout.
	cmd.Stderr = io.MultiWriter(errFile, &errb) // pack emits buildpack output to stderr.

	start := time.Now()
	t.Logf("Creating builder (logs %s)", filepath.Dir(outFile.Name()))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error creating builder: %v, logs:\nstdout: %s\nstderr:%s", err, outb.String(), errb.String())
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

// setupSource runs the given setup function to set up the source directory before a test.
func setupSource(t *testing.T, setup setupFunc, builder, src, app string) string {
	t.Helper()
	root := ""
	// Cloud Build runs in docker-in-docker mode where directories are mounted from the host daemon.
	// Therefore, we need to put the temporary directory in the shared /workspace volume.
	if cloudbuild {
		root = "/workspace"
	}
	temp, err := ioutil.TempDir(root, app)
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(temp) })
	sep := string(filepath.Separator)
	if _, err := runOutput("cp", "-R", src+sep+".", temp); err != nil {
		t.Fatalf("Error copying app files: %v", err)
	}
	if err := setup(builder, temp); err != nil {
		t.Fatalf("Error running test setup: %v", err)
	}
	return temp
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

func buildCommand(srcDir, image, builder string, env map[string]string, cache bool) []string {
	// Pack command to build app.
	args := strings.Fields(fmt.Sprintf("%s build %s --builder %s --path %s --pull-policy never --verbose --no-color --trust-builder", packBin, image, builder, srcDir))
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
	args = append(args, "--env", "GOOGLE_RANDOM="+randString(8), "--env", "GOOGLE_DEBUG=true")
	log.Printf("Running %v\n", args)
	return args
}

// buildApp builds an application image from source.
func buildApp(t *testing.T, srcDir, image, builder string, env map[string]string, cache bool, cfg Test) {
	t.Helper()

	attempts := cfg.FlakyBuildAttempts
	if attempts < 1 {
		attempts = 1
	}

	start := time.Now()
	var outb, errb bytes.Buffer

	for attempt := 1; attempt <= attempts; attempt++ {

		filename := fmt.Sprintf("%s-cache-%t", image, cache)
		if attempt > 1 {
			filename = fmt.Sprintf("%s-attempt-%d", filename, attempt)
		}
		outFile, errFile, cleanup := outFiles(t, builder, "pack-build", filename)
		defer cleanup()

		bcmd := buildCommand(srcDir, image, builder, env, cache)
		cmd := exec.Command(bcmd[0], bcmd[1:]...)
		cmd.Stdout = io.MultiWriter(outFile, &outb) // pack emits detect output to stdout.
		cmd.Stderr = io.MultiWriter(errFile, &errb) // pack emits build output to stderr.

		t.Logf("Building application %s (logs %s)", image, filepath.Dir(outFile.Name()))
		if err := cmd.Run(); err != nil {
			if attempt < attempts {
				t.Logf("Error building application %s, attempt %d of %d: %v, logs:\n%s\n%s", image, attempt, attempts, err, outb.String(), errb.String())
				outb.Reset()
				errb.Reset()
			} else {
				t.Fatalf("Error building application %s: %v, logs:\n%s\n%s", image, err, outb.String(), errb.String())
			}
		} else {
			// The application built successfully.
			break
		}
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
func buildFailingApp(t *testing.T, srcDir, image, builder string, env map[string]string) ([]byte, []byte, func()) {
	t.Helper()

	bcmd := buildCommand(srcDir, image, builder, env, false)
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
func verifyStructure(t *testing.T, image, builder string, cache bool, checks *StructureTest) {
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
		t.Fatalf("Error running structure tests: %v, logs:\n%s", err, outb.String())
	}
	t.Logf("Successfully ran structure tests on %s (in %s)", image, time.Since(start))
}

// verifyBuildMetadata verifies the image was built with correct buildpacks.
func verifyBuildMetadata(t *testing.T, image string, mustUse, mustNotUse []string, bom []BOMEntry) {
	t.Helper()

	start := time.Now()
	out, err := runOutput("docker", "inspect", "--format={{index .Config.Labels \"io.buildpacks.build.metadata\"}}", image)
	if err != nil {
		t.Fatalf("Error reading build metadata: %v", err)
	}

	var metadata struct {
		BOM        []BOMEntry `json:"bom"`
		Buildpacks []struct {
			ID string `json:"id"`
		} `json:"buildpacks"`
	}

	if err := json.Unmarshal([]byte(out), &metadata); err != nil {
		t.Fatalf("Error unmarshalling build metadata: %v", err)
	}

	usedBuildpacks := map[string]bool{}
	for _, bp := range metadata.Buildpacks {
		usedBuildpacks[bp.ID] = true
	}

	for _, id := range mustUse {
		if _, used := usedBuildpacks[id]; !used {
			t.Errorf("Must use buildpack %s was not used.", id)
		}
	}

	for _, id := range mustNotUse {
		if _, used := usedBuildpacks[id]; used {
			t.Errorf("Must not use buildpack %s was used.", id)
		}
	}

	if len(bom) != 0 {
		if got, want := metadata.BOM, bom; !reflect.DeepEqual(got, want) {
			t.Errorf("Unexpected BOM on image metadata\ngot: %v\nwant %v", got, want)
		}
	}

	t.Logf("Finished verifying build metadata (in %s)", time.Since(start))
}

// startContainer starts a container for the given app
// The function returns the containerID, the host and port at which the app is reachable and a cleanup function.
func startContainer(t *testing.T, image, entrypoint string, env []string, cache bool) (string, string, int, func()) {
	t.Helper()

	containerName := xid.New().String()
	command := []string{"docker", "run", "--detach", fmt.Sprintf("--name=%s", containerName)}
	for _, e := range env {
		command = append(command, "--env", e)
	}
	if cloudbuild {
		command = append(command, "--network=cloudbuild")
	} else {
		command = append(command, "--publish=8080")
	}
	if entrypoint != "" {
		command = append(command, "--entrypoint="+entrypoint)
	}
	command = append(command, image)

	id, err := runOutput(command...)
	if err != nil {
		t.Fatalf("Error starting container: %v", err)
	}
	t.Logf("Successfully started container: %s", id)

	host, port := getHostAndPortForApp(t, id, containerName)
	return id, host, port, func() {
		if _, err := runOutput("docker", "stop", id); err != nil {
			t.Logf("Failed to stop container: %v", err)
		}
		if t.Failed() {
			// output the container logs when a test failed, this can be useful for debugging failures in the test application
			outputDockerLogs(t, id)
		}
		if keepArtifacts {
			return
		}
		if _, err := runOutput("docker", "rm", "-f", id); err != nil {
			t.Logf("Failed to clean up container: %v", err)
		}
	}
}

func outputDockerLogs(t *testing.T, containerID string) {
	out, err := runDockerLogs(containerID, 1000)
	if err == nil {
		t.Logf("docker logs %v:\n%v", containerID, out)
	} else {
		t.Errorf("error fetching docker logs for container %v: %v", containerID, err)
	}
}

func getHostAndPortForApp(t *testing.T, containerID, containerName string) (string, int) {
	if cloudbuild {
		// In cloudbuild, the host environment is also a docker container and shares the same
		// network as the containers launched. In docker, within a network, the 'name' of a
		// container is also a hostname and can be used to address the container. This
		// useful for making sure http requests go to the intended application rather than
		// addressing by IP Address which, in the case of a terminated container, can lead to
		// addressing another newly started container which used the IP address of the
		// terminated container.
		//
		// Remapping random local ports to the container ports, like we do for the local test
		// runs, is not an option in cloudbuild because the build is run in a host docker
		// container and the launched containers are not "in" the host container and
		// therefore ports are not mapped at the build container's 'localhost'.
		return containerName, 8080
	}
	// When supplying the publish parameter with no local port picked, docker will
	// choose a random port to map to the published port. In this case, it means
	// there will be a mapping from localhost:${RAND_PORT} -> container:8080. There is a
	// small chance of a port collision. This can happen when we start a container and it
	// has a mapping from localhost:p1 -> container:8080, at this point the container
	// could terminate (example: app fails to start), another test could start a container
	// and docker could assign it the same random port 'p1'. The alternative is to write
	// our own port picker logic which would bring its own set of issues when run on
	// glinux machines which have various services running on ports. Since we have not
	// observed issues with the docker port picker, we are continuing to use it.
	return "localhost", hostPort(t, containerID)
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

// PullImages returns the value of the -pull-images flag.
func PullImages() bool {
	return pullImages
}
