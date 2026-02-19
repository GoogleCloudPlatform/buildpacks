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
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
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
	runImageOverride    string // Name of the run image to use during the test. This takes preference over the run-image defined in the builder.toml.
	builderPrefix       string // Prefix for created builder image.
	keepArtifacts       bool   // If true, keeps intermediate artifacts such as application images.
	packBin             string // Path to pack binary.
	structureBin        string // Path to container-structure-test binary.
	lifecycle           string // Path to lifecycle archive; optional.
	pullImages          bool   // Pull stack images instead of using local daemon.
	cloudbuild          bool   // Use cloudbuild network; required for Cloud Build.
	runtimeVersion      string // A runtime version which will be applied to tests that do not explicilty set a version.
	runtimeName         string // The name of the runtime (aka the language name such as 'go' or 'dotnet'). Used to properly set GOOGLE_RUNTIME.
	specialChars        = regexp.MustCompile("[^a-zA-Z0-9]+")
	buildEnv            string     // The build environment (dev, qual, or prod).
	containerEngine     = "docker" // containerEngine to use (docker or podman).
	remoteRepo          string     // Project ID to use for remote AR repository.
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
	flag.StringVar(&runImageOverride, "run-image-override", "", "Name of the run image to use during the test. This takes preference over the run-image defined in the builder.toml.")
	flag.StringVar(&builderPrefix, "builder-prefix", "acceptance-test-builder-", "Prefix for the generated builder image.")
	flag.BoolVar(&keepArtifacts, "keep-artifacts", false, "Keep images and other artifacts after tests have finished.")
	flag.StringVar(&packBin, "pack", "pack", "Path to pack binary.")
	flag.StringVar(&structureBin, "structure-test", "container-structure-test", "Path to container-structure-test.")
	flag.StringVar(&lifecycle, "lifecycle", "", "Location of lifecycle archive. Overrides builder.toml if specified.")
	flag.BoolVar(&pullImages, "pull-images", true, "Pull stack images before running the tests.")
	flag.BoolVar(&cloudbuild, "cloudbuild", false, "Use cloudbuild network; required for Cloud Build.")
	flag.StringVar(&runtimeVersion, "runtime-version", "", "A default runtime version which will be applied to the tests that do not explicitly set a version.")
	flag.StringVar(&runtimeName, "runtime-name", "", "The name of the runtime (aka the language name such as 'go' or 'dotnet'). Used to properly set GOOGLE_RUNTIME.")
	flag.StringVar(&buildEnv, "build-env", "", "The build environment to use (dev, qual, or prod). Sets GOOGLE_BUILD_ENV.")
	flag.StringVar(&containerEngine, "container-engine", "docker", "The container engine to use for running tests (docker or podman).")
	flag.StringVar(&remoteRepo, "remote-repo", "", "Project ID to use for remote AR repository.")
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
	// EnableCacheTest enables a second run of the test with the buildpacks cache enabled.
	EnableCacheTest bool
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
	// Map from label name to expected value.
	Labels map[string]string
	// Setup is a function that sets up the source directory before test.
	Setup setupFunc
	// VersionInclusionConstraint is a 'semver' inclusion filter for runtime versions. The FilterTest
	// method  will only return test cases with an inclusion constrant that matches with the value of the
	// `-runtime-version` flag. When the inclusion constraint or `runtime-version` flag are empty all
	// tests are included. See semver documentation to learn what is possible.
	VersionInclusionConstraint string
	// SkipStacks is slice of buildpack stack IDs that this test case should not be run on. This is
	// useful for excluding apps that do not compile on the min stack.
	SkipStacks []string
	// SkipPreReleaseVersions controls execution of the specified test case for pre-released versions.
	// i.e. rc and nightly candidates. If true the tests are skipped for the pre-released versions
	SkipPreReleaseVersions bool
}

// SetupContext is passed into the Test.Setup function, it gives the setupFunc implementor access
// to various fields to determine what modifications they should make to their source.
type SetupContext struct {
	// SrcDir contains a path to a modifable copy of the source on local disk that will be copied
	// to the build environment at /workspace.
	SrcDir string
	// Builder is the name of the builder image.
	Builder string
	// RuntimeVersion is the version for which this test run will be performed.
	RuntimeVersion string
}

// ImageContext holds information about the buildpack stack images used for a test. It is returned
// by ProvisionImages and must be passed as an argument to TestApp and TestBuildFailure.
type ImageContext struct {
	// The ID of the builpack stack.
	StackID string
	// The builder image name.
	BuilderImage string
	// The run image name.
	RunImage string
}

// setupFunc is a function that is called before the test starts and can be used to modify the test source.
// The setupCtx.SrcDir property contains a path to a copy of the source which can be modified before the
// test runs.
type setupFunc func(setupCtx SetupContext) error

// builderTOML contains the values from builder.toml file
type builderTOML struct {
	Stack struct {
		ID         string `toml:"id"`
		RunImage   string `toml:"run-image"`
		BuildImage string `toml:"build-image"`
	} `toml:"stack"`
}

func constructImageName(t *testing.T, name, builderName string) string {
	t.Helper()
	image := fmt.Sprintf("%s-%s", strings.ToLower(specialChars.ReplaceAllString(name, "-")), builderName)
	if remoteRepo != "" {
		return fmt.Sprintf("us-docker.pkg.dev/%s/acceptance-tests/%s", remoteRepo, image)
	}
	return image
}

// TestApp builds and a single application and verifies that it runs and handles requests.
func TestApp(t *testing.T, imageCtx ImageContext, cfg Test) {
	t.Helper()

	env := prepareEnvTest(t, cfg)

	if cfg.Name == "" {
		cfg.Name = cfg.App
	}

	// Docker image names may not contain underscores or start with a capital letter.
	builderName := imageCtx.BuilderImage
	runName := imageCtx.RunImage
	image := constructImageName(t, cfg.Name, builderName)

	// Delete the docker image and volumes created by pack during the build.
	defer func() {
		if remoteRepo == "" {
			cleanUpVolumes(t, image)
		}
		cleanUpImage(t, image)
	}()

	// Create a configuration for container-structure-tests.
	checks := NewStructureTest(cfg.FilesMustExist, cfg.FilesMustNotExist)

	// Run Setup function if provided.
	src := filepath.Join(testData, cfg.App)
	if cfg.Setup != nil {
		src = setupSource(t, cfg.Setup, builderName, src, cfg.App)
	}

	if cfg.EnableCacheTest {
		testAppWithCache(t, src, image, builderName, runName, env, checks, cfg)
	} else {
		testApp(t, src, image, builderName, runName, env, false, checks, cfg)
	}
}

func testAppWithCache(t *testing.T, src, image, builderName, runName string, env map[string]string, checks *StructureTest, cfg Test) {
	// Run a no-cache build, followed by a cache build
	t.Run("cache false", func(t *testing.T) {
		testApp(t, src, image, builderName, runName, env, false, checks, cfg)
	})
	t.Run("cache true", func(t *testing.T) {
		testApp(t, src, image, builderName, runName, env, true, checks, cfg)
	})
}

func testApp(t *testing.T, src, image, builderName, runName string, env map[string]string, cacheEnabled bool, checks *StructureTest, cfg Test) {
	buildApp(t, src, image, builderName, runName, env, cacheEnabled, cfg)
	verifyBuildMetadata(t, image, cfg.MustUse, cfg.MustNotUse)
	verifyLabelValues(t, image, cfg.Labels)
	verifyStructure(t, image, builderName, cacheEnabled, checks)
	invokeApp(t, cfg, image, cacheEnabled)
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
	// VersionInclusionConstraint is a 'semver' inclusion filter for runtime versions. The FilterTest
	// method  will only return test cases with an inclusion constrant that matches with the value of the
	// `-runtime-version` flag. When the inclusion constraint or `runtime-version` flag are empty all
	// tests are included. See semver documentation to learn what is possible.
	VersionInclusionConstraint string
	// SkipPreReleaseVersions controls execution of the specified test case for pre-released versions.
	// i.e. rc and nightly candidates. If true the tests are skipped for the pre-released versions
	SkipPreReleaseVersions bool
}

// TestBuildFailure runs a build and ensures that it fails. Additionally, it ensures the emitted logs match mustMatch regexps.
func TestBuildFailure(t *testing.T, imageCtx ImageContext, cfg FailureTest) {
	t.Helper()

	env := prepareEnvFailureTest(t, cfg)

	if cfg.Name == "" {
		cfg.Name = cfg.App
	}
	builderName := imageCtx.BuilderImage
	runName := imageCtx.RunImage
	image := constructImageName(t, cfg.Name, builderName)

	// Delete the docker volumes created by pack during the build.
	if !keepArtifacts && remoteRepo == "" {
		defer cleanUpVolumes(t, image)
	}

	src := filepath.Join(testData, cfg.App)
	if cfg.Setup != nil {
		src = setupSource(t, cfg.Setup, builderName, src, cfg.App)
	}

	// combinedb is the combined output of the build i.e. it includes stdout and stderr.
	combinedb, cleanup := buildFailingApp(t, src, image, builderName, runName, env)
	defer cleanup()

	r, err := regexp.Compile(cfg.MustMatch)
	if err != nil {
		t.Fatalf("regexp %q failed to compile: %v", r, err)
	}
	if r.Match(combinedb) {
		t.Logf("Expected regex %q found in combined (stdout + stderr).", r)
	} else {
		t.Errorf("Expected regex %q not found in combined (stdout + stderr).\n\n--- COMBINED OUTPUT ---\n%s\n--- END OUTPUT ---", r, combinedb)
	}
	expectedLog := "Expected pattern included in error output: true"
	builderOutput := string(combinedb)
	if !cfg.SkipBuilderOutputMatch && !strings.Contains(builderOutput, expectedLog) {
		t.Errorf("Expected regexp %q not found in BUILDER_OUTPUT", r)
		t.Logf("BUILDER_OUTPUT: %v", builderOutput)
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
		if !strings.Contains(body, cfg.MustMatch) {
			t.Errorf("Response body does not contain: got %q, want %q", body, cfg.MustMatch)
		}
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

// runContainerLogs returns the logs for a container, the lineLimit parameter
// controls the maximum number of lines read from the log.
func runContainerLogs(containerID string, lineLimit int) (string, error) {
	return runCombinedOutput(containerEngine, "logs", "--tail", fmt.Sprint(lineLimit), containerID)
}

// cleanUpImage attempts to delete an image from the Docker daemon.
func cleanUpImage(t *testing.T, name string) {
	t.Helper()

	if keepArtifacts {
		return
	}
	if _, err := runOutput(containerEngine, "rmi", "-f", name); err != nil {
		t.Logf("Failed to clean up image: %v", err)
	}
}

// ProvisionImages provisions the builder, build, and run images necessary for running
// a test.
//
// The 'builderName' return value is the name of the builder image.
// The 'runName' return value is the name of the run image. This value will be the
// empty string when the run image is not override and the builder's default run
// image is to be used.
//
// The 'cleanup' return value is a function which should be run after the tests are
// complete to clean up the images which are explicitly created. Images that are
// pulled are not cleaned up to prevent conflicts with other tests.
func ProvisionImages(t *testing.T) (ImageContext, func()) {
	t.Helper()

	if err := checktools.Installed(); err != nil {
		t.Fatalf("Error checking tools: %v", err)
	}
	if err := checktools.PackVersion(); err != nil {
		t.Fatalf("Error checking pack version: %v", err)
	}

	builderName := generateRandomImageName(builderPrefix)
	if builderImage != "" {
		t.Logf("Testing existing builder image: %s", builderImage)
		if pullImages {
			if _, err := runOutput(containerEngine, "pull", builderImage); err != nil {
				t.Fatalf("Error pulling %s: %v", builderImage, err)
			}
		}
		// Pack cache is based on builder name; retag with a unique name.
		if _, err := runOutput(containerEngine, "tag", builderImage, builderName); err != nil {
			t.Fatalf("Error tagging %s as %s: %v", builderImage, builderName, err)
		}
		runName, cleanUpRun, err := provisionRunImageFromBuilder(builderName)
		if err != nil {
			t.Fatalf("Error provisioning run image for builder %q: %v", builderName, err)
		}
		stackID, err := getImageStackID(builderName)
		if err != nil {
			t.Fatalf("Getting stack ID from builder %q: %v", builderName, err)
		}
		imageCtx := ImageContext{
			StackID:      stackID,
			BuilderImage: builderName,
			RunImage:     runName,
		}
		return imageCtx, func() {
			cleanUpImage(t, builderName)
			cleanUpRun(t)
		}
	}

	builderLoc, cleanUpBuilder := extractBuilder(t, builderSource)

	// For language builders, the toml file is named as builder.toml.
	// For universal builders, the toml file is named like google.24.builder.toml.
	files, err := os.ReadDir(builderLoc)
	if err != nil {
		t.Fatalf("Error reading builder location: %v", err)
	}
	var config string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), "builder.toml") {
			config = filepath.Join(builderLoc, f.Name())
			break
		}
	}
	if config == "" {
		t.Fatalf("No builder.toml file found in %s", builderLoc)
	}

	if lifecycle != "" {
		t.Logf("Using lifecycle location: %s", lifecycle)
		if c, err := updateLifecycle(config, lifecycle); err != nil {
			t.Fatalf("Error updating lifecycle location: %v", err)
		} else {
			config = c
		}
	}

	builderConfig, err := readBuilderTOML(config)
	if err != nil {
		t.Fatalf("Error reading builder.toml: %v", err)
	}
	// Pull images once in the beginning to prevent them from changing in the middle of testing.
	// The images are intentionally not cleaned up to prevent conflicts across different test targets.
	if pullImages {
		buildName := builderConfig.Stack.BuildImage
		if _, err := runOutput(containerEngine, "pull", buildName); err != nil {
			t.Fatalf("Error pulling %s: %v", buildName, err)
		}
	}
	runName, cleanUpRun, err := provisionRunImageFromTOML(builderConfig)
	if err != nil {
		t.Fatalf("Error provisioning run image: %v", err)
	}

	// Pack command to create the builder.
	args := strings.Fields(fmt.Sprintf("builder create %s --config %s --pull-policy never --verbose --no-color", builderName, config))
	cmd := exec.Command(packBin, args...)

	outFile, errFile, cleanup := outFiles(t, builderName, "pack", "create-builder")
	defer cleanup()
	// Newer versions of the CNB lifecycle (v0.20.13+) send the stdout and
	// stderr of the executing command to the same stream (stdout), while
	// older versions send them to separate streams.
	var combinedb bytes.Buffer
	cmd.Stdout = io.MultiWriter(outFile, &combinedb)
	cmd.Stderr = io.MultiWriter(errFile, &combinedb)

	start := time.Now()
	t.Logf("Creating builder (logs %s)", filepath.Dir(outFile.Name()))

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create builder: %s\nerror: %v\noutput:\n%s", cmd.String(), err, combinedb.String())
	}

	t.Logf("Successfully created builder: %s (in %s)", builderName, time.Since(start))

	imageCtx := ImageContext{
		StackID:      builderConfig.Stack.ID,
		BuilderImage: builderName,
		RunImage:     runName,
	}

	return imageCtx, func() {
		cleanUpImage(t, builderName)
		cleanUpBuilder()
		cleanUpRun(t)
	}
}

func provisionRunImageFromTOML(builderConfig *builderTOML) (string, func(t *testing.T), error) {
	runName := builderConfig.Stack.RunImage
	if runImageOverride != "" {
		runName = runImageOverride
	}
	if pullImages {
		if _, err := runOutput(containerEngine, "pull", runName); err != nil {
			return "", nil, fmt.Errorf("pulling %q: %w", runName, err)
		}
	}
	if runName == builderConfig.Stack.RunImage {
		// when the run image name is the one defined in the builderconfig, do not verify the stack ids
		// match because a builder.toml should contain valid configuration.
		return runName, func(t *testing.T) {}, nil
	}
	return provisionImageWithMatchingStackID(runName, builderConfig.Stack.ID)
}

func provisionRunImageFromBuilder(builderName string) (string, func(t *testing.T), error) {
	builderDefinedRunImage, err := runImageFromMetadata(builderName)
	if err != nil {
		return "", nil, fmt.Errorf("Error extracting run image from image %q: %w", builderName, err)
	}
	runName := builderDefinedRunImage
	if runImageOverride != "" {
		runName = runImageOverride
	}
	if pullImages {
		if _, err := runOutput(containerEngine, "pull", runName); err != nil {
			return "", nil, fmt.Errorf("pulling %q: %w", runName, err)
		}
	}
	if builderDefinedRunImage == runName {
		// when the run image is the one defined for the builder, do not verify the stack ids match
		// because the builder should contain valid configuration.
		return runName, func(t *testing.T) {}, nil
	}
	builderStackID, err := getImageStackID(builderName)
	if err != nil {
		return "", nil, fmt.Errorf("getting stack id of builder %q: %w", builderName, err)
	}
	return provisionImageWithMatchingStackID(runName, builderStackID)
}

// provisionImageWithMatchingStackId returns an image with the contents of 'fromImage' and a stack
// ID of 'stackID'. The second return value is a cleanUp function which will destroy the returned
// image if the image was newly created. The cleanUp function is a no-op if the fromImage already
// had the desired stackID.
//
// This function is useful for ensuring a run image has the same stack id as the builder. This is
// necessary because pack requires that the two match.
func provisionImageWithMatchingStackID(fromImage, stackID string) (string, func(t *testing.T), error) {
	imageStackID, err := getImageStackID(fromImage)
	if err != nil {
		return "", nil, fmt.Errorf("getting stack id of image %q: %w", fromImage, err)
	}
	if imageStackID == stackID {
		return fromImage, func(t *testing.T) {}, nil
	}
	newImage, err := newImageWithStackID(fromImage, stackID)
	if err != nil {
		return "", nil, fmt.Errorf("creating image from %q with stack id %q: %w", fromImage, stackID, err)
	}
	cleanUp := func(t *testing.T) {
		cleanUpImage(t, newImage)
	}
	return newImage, cleanUp, nil
}

func getImageStackID(image string) (string, error) {
	out, err := runOutput(containerEngine, "inspect", `--format={{index .Config.Labels "io.buildpacks.stack.id"}}`, image)
	if err != nil {
		return "", fmt.Errorf("getting stack id from %s inspect: %w", containerEngine, err)
	}
	return out, nil
}

func newImageWithStackID(fromImage, stackID string) (string, error) {
	newImage := generateRandomImageName(fromImage)
	_, err := runCombinedOutput("bash", "-c", fmt.Sprintf(`echo "FROM %s" | %s build --label io.buildpacks.stack.id="%s" -t "%s" -`, fromImage, containerEngine, stackID, newImage))
	if err != nil {
		return "", fmt.Errorf("changing stack id label on %q: %v", fromImage, err)
	}
	return newImage, nil
}

func generateRandomImageName(baseName string) string {
	rand := randString(10)
	if strings.Contains(baseName, ":") {
		return fmt.Sprintf("%v_%v", baseName, rand)
	}
	return baseName + rand
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
	out, err := runOutput(containerEngine, "inspect", image, format)
	if err != nil {
		return "", fmt.Errorf("reading builder metadata: %v", err)
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
	temp, err := ioutil.TempDir(root, path.Base(app))
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(temp) })
	sep := string(filepath.Separator)
	if _, err := runOutput("cp", "-R", src+sep+".", temp); err != nil {
		t.Fatalf("Error copying app files: %v", err)
	}
	setupCtx := SetupContext{
		SrcDir:         temp,
		Builder:        builder,
		RuntimeVersion: runtimeVersion,
	}
	if err := setup(setupCtx); err != nil {
		t.Fatalf("Error running test setup: %v", err)
	}
	return temp
}

func readBuilderTOML(path string) (*builderTOML, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	var bc builderTOML
	if err := toml.Unmarshal(bytes, &bc); err != nil {
		return nil, fmt.Errorf("unmarshalling %q: %w", path, err)
	}
	return &bc, nil
}

func buildCommand(srcDir, image, builderName, runName string, env map[string]string, cache bool) []string {
	// Pack command to build app.
	cmd := fmt.Sprintf("%s build %s --builder %s --path %s --pull-policy never --verbose --no-color --trust-builder %s", packBin, image, builderName, srcDir, cacheOptions(image))
	if remoteRepo != "" {
		cmd += " --publish"
	}
	args := strings.Fields(cmd)
	if runName != "" {
		args = append(args, "--run-image", runName)
		if hasRuntimePreinstalled(runName) {
			// This skips adding the language runtime downloaded during the build to the final
			// image as a launch layer. For non-generic run images, the language runtime is already
			// included in the base image.
			args = append(args, "--env", "X_GOOGLE_SKIP_RUNTIME_LAUNCH=true")
		}
	}
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
	args = append(args, "--network", "host")
	args = append(args, "--env", "GOOGLE_RUNTIME_IMAGE_REGION=us")
	if buildEnv != "" {
		args = append(args, "--env", fmt.Sprintf("GOOGLE_BUILD_ENV=%s", buildEnv))
	}
	log.Printf("Running %v\n", args)
	return args
}

func cacheOptions(image string) string {
	if remoteRepo != "" {
		return fmt.Sprintf("--cache type=build;format=image;name=%s.build --cache type=launch;format=image;name=%s.launch", image, image)
	}
	buildVolume, launchVolume := volumeNames(image)
	return fmt.Sprintf("--cache type=build;format=volume;name=%s --cache type=launch;format=volume;name=%s", buildVolume, launchVolume)
}

// hasRuntimePreinstalled returns whether or not the image is the "generic" run image that does not
// contain the language runtime built in. For containers built on the generic run image, the
// language runtime is added dynamically during the build instead. The OSS builder and GAE Flex
// build on the generic run images. GCF and GAE standard use language-specific run images to allow
// the language runtime to be updated during automatic base image updates.
func hasRuntimePreinstalled(runName string) bool {
	// Generic run image example (should NOT match):
	// gcr.io/gae-runtimes/buildpacks/google-gae-22/nodejs/run
	// gcr.io/gae-runtimes/stacks/google-gae-18/run
	// gcr.io/buildpacks/google-18/run
	//
	// Non-generic run image example (should match):
	// gcr.io/gae-runtimes/buildpacks/nodejs14/run
	// gcr.io/${PROJECT}/buildpacks/${RUNTIME}/run
	re := regexp.MustCompile(`/buildpacks/(?:go|nodejs|dotnet|java|php|ruby|python)\d+/run`)
	return re.MatchString(runName)
}

// buildApp builds an application image from source.
func buildApp(t *testing.T, srcDir, image, builderName, runName string, env map[string]string, cache bool, cfg Test) {
	t.Helper()

	attempts := cfg.FlakyBuildAttempts
	if attempts < 1 {
		attempts = 1
	}

	start := time.Now()
	var combinedb bytes.Buffer

	for attempt := 1; attempt <= attempts; attempt++ {

		filename := fmt.Sprintf("%s-cache-%t", image, cache)
		if attempt > 1 {
			filename = fmt.Sprintf("%s-attempt-%d", filename, attempt)
		}
		outFile, errFile, cleanup := outFiles(t, builderName, "pack-build", filename)
		defer cleanup()

		bcmd := buildCommand(srcDir, image, builderName, runName, env, cache)
		cmd := exec.Command(bcmd[0], bcmd[1:]...)

		// Newer versions of the CNB lifecycle (v0.20.13+) send the stdout and
		// stderr of the executing command to the same stream (stdout), while
		// older versions send them to separate streams.
		cmd.Stdout = io.MultiWriter(outFile, &combinedb)
		cmd.Stderr = io.MultiWriter(errFile, &combinedb)

		t.Logf("Building application %s (logs %s)", image, filepath.Dir(outFile.Name()))
		if err := cmd.Run(); err != nil {
			if attempt < attempts {
				t.Logf("Error building application %s, attempt %d of %d: %v, logs:\n%s", image, attempt, attempts, err, combinedb.String())
				combinedb.Reset()
			} else {
				t.Fatalf("Error building application %s: %v, logs:\n%s", image, err, combinedb.String())
			}
		} else {
			t.Logf("Successfully built application %s: %v, logs:\n%s", image, err, combinedb.String())
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
		if !strings.Contains(combinedb.String(), text) {
			t.Errorf("Build logs must contain %q:\n%s", text, combinedb.String())
		}
	}
	for _, text := range mustNotOutput {
		if strings.Contains(combinedb.String(), text) {
			t.Errorf("Build logs must not contain %q:\n%s", text, combinedb.String())
		}
	}

	// Scan for incorrect cache hits/misses.
	if cache {
		if strings.Contains(combinedb.String(), cacheMissMessage) {
			t.Fatalf("FAIL: Cached build had a cache miss:\n%s", combinedb.String())
		}
	} else {
		if strings.Contains(combinedb.String(), cacheHitMessage) {
			t.Fatalf("FAIL: Non-cache build had a cache hit:\n%s", combinedb.String())
		}
	}

	t.Logf("Successfully built application: %s (in %s)", image, time.Since(start))
}

// buildFailingApp attempts to build an app and ensures that it fails (non-zero exit code).
// It returns the interleaved stdout and stderr (combined chronologically) and a cleanup function.
// Merging these streams ensures that buildpack error messages are captured regardless of whether
// they are emitted to stdout or stderr, which is required for compatibility Lifecycle versions
// v0.20.13+.
func buildFailingApp(t *testing.T, srcDir, image, builderName, runName string, env map[string]string) ([]byte, func()) {
	t.Helper()

	bcmd := buildCommand(srcDir, image, builderName, runName, env, false)
	cmd := exec.Command(bcmd[0], bcmd[1:]...)

	outFile, errFile, cleanup := outFiles(t, builderName, "pack-build-failing", image)
	defer cleanup()
	var combinedb bytes.Buffer
	// Newer versions of the CNB lifecycle (v0.20.13+) send the stdout and
	// stderr of the executing command to the same stream (stdout), while
	// older versions send them to separate streams.
	cmd.Stdout = io.MultiWriter(outFile, &combinedb)
	cmd.Stderr = io.MultiWriter(errFile, &combinedb)

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

	return combinedb.Bytes(), func() {
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

func verifyLabelValues(t *testing.T, image string, labels map[string]string) {
	t.Helper()

	start := time.Now()

	for label, value := range labels {
		out, err := runOutput("docker", "inspect", fmt.Sprintf("--format={{index .Config.Labels %q}}", label), image)
		if err != nil {
			t.Errorf("Error reading label %v: %v", label, err)
		} else if out != value {
			t.Errorf("Unexpected value for label %v\ngot: %v\nwant %v", label, out, value)
		}
	}

	t.Logf("Finished verifying label values (in %s)", time.Since(start))
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

	t.Logf("Finished verifying build metadata (in %s)", time.Since(start))
}

// startContainer starts a container for the given app
// The function returns the containerID, the host and port at which the app is reachable and a cleanup function.
func startContainer(t *testing.T, image, entrypoint string, env []string, cache bool) (string, string, int, func()) {
	t.Helper()

	if remoteRepo != "" {
		if _, err := runOutput(containerEngine, "pull", image); err != nil {
			t.Fatalf("Pulling image %q: %v", image, err)
		}
	}

	containerName := xid.New().String()
	command := []string{containerEngine, "run", "--detach", fmt.Sprintf("--name=%s", containerName)}
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
		if _, err := runOutput(containerEngine, "stop", id); err != nil {
			t.Logf("Failed to stop container: %v", err)
		}
		if t.Failed() {
			// output the container logs when a test failed, this can be useful for debugging failures in the test application
			outputContainerLogs(t, id)
		}
		if keepArtifacts {
			return
		}
		if _, err := runOutput(containerEngine, "rm", "-f", id); err != nil {
			t.Logf("Failed to clean up container: %v", err)
		}
	}
}

func outputContainerLogs(t *testing.T, containerID string) {
	out, err := runContainerLogs(containerID, 1000)
	if err == nil {
		t.Logf("%s logs %v:\n%v", containerEngine, containerID, out)
	} else {
		t.Errorf("error fetching %s logs for container %v: %v", containerEngine, containerID, err)
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
	if v := os.Getenv("DOCKER_IP_UBUNTU"); v != "" {
		return v, hostPort(t, containerID)
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
	portstr, err := runOutput(containerEngine, "inspect", id, format)
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

	buildVolume, launchVolume := volumeNames(image)
	if _, err := runOutput(containerEngine, "volume", "rm", "-f", launchVolume, buildVolume); err != nil {
		t.Logf("Failed to clean up cache volumes: %v", err)
	}
}

func volumeNames(image string) (string, string) {
	// This logic is copied from pack's codebase. See:
	// https://github.com/buildpacks/pack/blob/92bc87b297695e4ac6baf559bad2efd55aecec1f/internal/paths/paths.go#L81
	reservedNameConversions := map[string]string{
		"aux": "a_u_x",
		"com": "c_o_m",
		"con": "c_o_n",
		"lpt": "l_p_t",
		"nul": "n_u_l",
		"prn": "p_r_n",
		":":   "_",
	}
	for k, v := range reservedNameConversions {
		image = strings.ReplaceAll(image, k, v)
	}
	return image + ".build", image + ".launch"
}

// PullImages returns the value of the -pull-images flag.
func PullImages() bool {
	return pullImages
}

// FilterTests returns a new slice with only tests that should be run. Tests are filtered out if
// their VersionInclusionConstraint does not match the `-runtime-version` flag.
func FilterTests(t *testing.T, imageCtx ImageContext, testCases []Test) []Test {
	results := make([]Test, 0)
	for _, tc := range testCases {
		if tc.SkipPreReleaseVersions && isPreReleaseVersion() {
			continue
		}
		if ShouldTestVersion(t, tc.VersionInclusionConstraint) && ShouldTestStack(t, imageCtx.StackID, tc.SkipStacks) {
			results = append(results, tc)
		}
	}
	return results
}

// FilterFailureTests returns a new slice with only tests that should be run. Tests are filtered out
// if their VersionInclusionConstraint does not match the `-runtime-version` flag.
func FilterFailureTests(t *testing.T, testCases []FailureTest) []FailureTest {
	results := make([]FailureTest, 0)
	for _, tc := range testCases {
		if tc.SkipPreReleaseVersions && isPreReleaseVersion() {
			continue
		}
		if ShouldTestVersion(t, tc.VersionInclusionConstraint) {
			results = append(results, tc)
		}
	}
	return results
}

// isPreReleaseVersion returns true if the runtime version is a pre-release version.
func isPreReleaseVersion() bool {
	return strings.Contains(runtimeVersion, "rc") || strings.Contains(runtimeVersion, "nightly") || strings.Contains(runtimeVersion, "RC")
}

// ShouldTestStack returns true if the current test should be included on test runs using the given
// buildpack stack.
func ShouldTestStack(t *testing.T, stackID string, skipStacks []string) bool {
	t.Helper()
	for _, skipStack := range skipStacks {
		if skipStack == stackID {
			return false
		}
	}
	return true
}

// ShouldTestVersion returns true if the current test run's version is included
// in the constraint parameter. An empty inclusion constraint is treated as
// matching all versions.
//
// The version comparison check supports partial matches. For example, an excluded
// version of '12.5' will match all '12.5.x' versions. In addition, you can specify
// ranges such as '>=10.0.0'. The version comparison uses semver2 for the constraint
// comparision. See the documentation for semver2 to learn more.
func ShouldTestVersion(t *testing.T, inclusionConstraint string) bool {
	t.Helper()

	v := runtimeVersion
	if v == "" || inclusionConstraint == "" {
		return true
	}
	// The format of Go pre-release version e.g. 1.20rc1 doesn't follow the semver rule
	// that requires a hyphen before the identifier "rc".
	if strings.Contains(v, "rc") && !strings.Contains(v, "-rc") {
		v = strings.Replace(v, "rc", "-rc", 1)
	}
	if strings.Contains(v, "RC") && !strings.Contains(v, "-RC") {
		v = strings.Replace(v, "RC", "-RC", 1)
	}
	// The format of Java candidates such as 23.0.1_11 needs to be converted to valid semver 23.0.1+11
	if strings.HasPrefix(runtimeName, "java") && strings.Contains(v, "_") {
		v = strings.Replace(v, "_", "+", 1)
	}

	re := regexp.MustCompile(`(?i)[-.]?rc.*`)
	v = re.ReplaceAllString(v, "")
	rtVer, err := semver.NewVersion(v)
	if err != nil {
		t.Fatalf("Unable to use %q as a semver.Version: %v", v, err)
	}
	return versionMatches(t, rtVer, inclusionConstraint)
}

func versionMatches(t *testing.T, version *semver.Version, constraint string) bool {
	t.Helper()
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		t.Fatalf("Unable to use %q as a semver.Constraint: %v", constraint, err)
	}
	return c.Check(version)
}

func sliceContains(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func getNewVersions(runtime string) []string {
	// Check if JSON file exists.
	if _, err := os.Stat("new_versions.json"); err != nil {
		return nil
	}
	versionMap := make(map[string][]string)
	file, err := ioutil.ReadFile("new_versions.json")
	if err != nil {
		log.Fatalf("Error parsing JSON file: %q", err)
	}
	if err = json.Unmarshal(file, &versionMap); err != nil {
		log.Fatalf("Unable to decode JSON version map: %q", err)
	}
	if _, ok := versionMap[runtime]; !ok {
		return nil
	}
	return versionMap[runtime]
}
