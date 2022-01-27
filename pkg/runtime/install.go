// Copyright 2022 Google LLC
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

package runtime

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/buildpacks/libcnb"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	dartSdkURL     = "https://storage.googleapis.com/dart-archive/channels/stable/release/%[1]s/sdk/dartsdk-linux-x64-release.zip"
	rubyRuntimeURL = "https://dl.google.com/runtimes/ruby/ruby-%s.tar.gz"
)

const (
	versionKey = "version"
	// gcpUserAgent is required for the Ruby runtime, but used for others for simplicity.
	gcpUserAgent = "GCPBuildpacks"
)

// IsCached returns true if the requested version of a runtime is installed in the given layer.
func IsCached(ctx *gcp.Context, layer *libcnb.Layer, version string) bool {
	metaVersion := ctx.GetMetadata(layer, versionKey)
	return metaVersion == version
}

// InstallDartSDK downloads a given version of the dart SDK to the specified layer.
func InstallDartSDK(ctx *gcp.Context, layer *libcnb.Layer, version string) error {
	ctx.ClearLayer(layer)
	sdkURL := fmt.Sprintf(dartSdkURL, version)

	zip, err := ioutil.TempFile(layer.Path, "dart-sdk-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(zip.Name())

	if err := fetchRuntime("Dart SDK", version, sdkURL, zip); err != nil {
		return err
	}

	if _, err := ctx.ExecWithErr([]string{"unzip", "-q", zip.Name(), "-d", layer.Path}); err != nil {
		return fmt.Errorf("extracting Dart SDK: %v", err)
	}

	// Once extracted the SDK contents are in a subdirectory called "dart-sdk". We move everything up
	// one level so "bin" and "lib" end up in the layer path.
	files, err := ioutil.ReadDir(path.Join(layer.Path, "dart-sdk"))
	if err != nil {
		return err
	}
	for _, file := range files {
		op := path.Join(layer.Path, "dart-sdk", file.Name())
		np := path.Join(layer.Path, file.Name())
		if err := os.Rename(op, np); err != nil {
			return err
		}
	}

	ctx.SetMetadata(layer, versionKey, version)

	return nil
}

// InstallRuby downloads a given version of the ruby runtime to the specified layer.
func InstallRuby(ctx *gcp.Context, layer *libcnb.Layer, version string) error {
	ctx.ClearLayer(layer)
	runtimeURL := fmt.Sprintf(rubyRuntimeURL, version)

	tar, err := ioutil.TempFile(layer.Path, "ruby-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tar.Name())

	if err := fetchRuntime("Ruby Runtime", version, runtimeURL, tar); err != nil {
		return err
	}

	if _, err := ctx.ExecWithErr([]string{"tar", "-xzvf", tar.Name(), "--directory", layer.Path}); err != nil {
		return fmt.Errorf("extracting Ruby Runtime: %v", err)
	}

	ctx.SetMetadata(layer, versionKey, version)

	return nil
}

// fetchRuntime downloads a runtime archive from the given URL and writes it to the given file.
func fetchRuntime(name, version, url string, f io.Writer) error {
	client := newRetryableHTTPClient()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return downloadError(name, version, url, err)
	}

	req.Header.Set("User-Agent", gcpUserAgent)

	response, err := client.Do(req)
	if err != nil {
		return downloadError(name, version, url, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return downloadError(name, version, url, fmt.Errorf("HTTP status: %d", response.StatusCode))
	}

	if _, err = io.Copy(f, response.Body); err != nil {
		return fmt.Errorf("copying %s: %v", name, err)
	}

	return nil
}

// downloadError returns an user error that includes instructions about how to correctly specify
// a valid runtime version.
func downloadError(name, version, url string, err error) error {
	return gcp.UserErrorf("%s version %s does not exist at %s (%v). You can specify the version by setting the %s environment variable.", name, version, url, err, env.RuntimeVersion)
}

// newRetryableHTTPClient configures an http client for automatic retries.
func newRetryableHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	return retryClient.StandardClient()
}
