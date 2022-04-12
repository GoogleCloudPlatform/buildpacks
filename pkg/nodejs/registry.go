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

package nodejs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/hashicorp/go-retryablehttp"
)

// npmRegistryURL responds with the registry metadata for a given NPM package.
var npmRegistryURL = "https://registry.npmjs.org/%s"

// yarnTagsURL responds with all available Yarn versions >=2.0.0.
var yarnTagsURL = "https://repo.yarnpkg.com/tags"

// packageMetadata contains registry information about an npm package. For more information see
// https://github.com/npm/registry/blob/master/docs/responses/package-metadata.md#abbreviated-metadata-format.
type packageMetadata struct {
	Name     string `json:"name"`
	Modified string `json:"modified"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Versions map[string]interface{} `json:"versions"`
}

type yarnTags struct {
	Latest struct {
		Stable string `json:"stable"`
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Tags []string `json:"tags"`
}

// latestPackageVersion returns latest available verion of an NPM package.
func latestPackageVersion(pkg string) (string, error) {
	metadata, err := fetchPackageMetadata(pkg)
	if err != nil {
		return "", err
	}
	return metadata.DistTags.Latest, nil
}

// resolvePackageVersion returns the newest available version of an NPM package that satisfies the
// provided version constraint.
func resolvePackageVersion(pkg, verConstraint string) (string, error) {
	if version.IsExactSemver(verConstraint) {
		return verConstraint, nil
	}

	metadata, err := fetchPackageMetadata(pkg)
	if err != nil {
		return "", err
	}

	// After v2 yarn stopped publishing packages to the NPM registry so we need to fetch these from
	// elsewhere. We do not include the additional versions for the * or "" wildcard versions to avoid
	// breaking customers implicitly pinned to v1.x.x.
	var extraVersions []string
	if pkg == "yarn" && verConstraint != "" && verConstraint != "*" {
		extraVersions, err = fetchYarnTags()
		if err != nil {
			return "", err
		}
	}

	versions := make([]string, 0, len(metadata.Versions)+len(extraVersions))
	for v := range metadata.Versions {
		versions = append(versions, v)
	}
	for _, v := range extraVersions {
		versions = append(versions, v)
	}

	return version.ResolveVersion(verConstraint, versions)
}

// fetchYarnTags fetches metadata available Yarn versions >=2.0.0.
func fetchYarnTags() ([]string, error) {
	bytes, err := sendRequest(yarnTagsURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("getting url %q: %w", yarnTagsURL, err)
	}

	var tags yarnTags
	if err := json.Unmarshal(bytes, &tags); err != nil {
		return nil, fmt.Errorf("unmarshalling response from %q: %w", yarnTagsURL, err)
	}
	return tags.Tags, nil
}

// fetchPackageMetadata fetches metadata about an NPM package published to the NPM registry.
func fetchPackageMetadata(pkg string) (*packageMetadata, error) {
	url := fmt.Sprintf(npmRegistryURL, pkg)
	header := http.Header{
		"Accept": []string{"application/vnd.npm.install-v1+json"},
	}

	bytes, err := sendRequest(url, header)
	if err != nil {
		return nil, fmt.Errorf("getting url %q: %w", url, err)
	}

	var metadata packageMetadata
	if err := json.Unmarshal(bytes, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshalling response from %q: %w", url, err)
	}
	return &metadata, nil
}

func sendRequest(url string, header http.Header) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %q: %w", url, err)
	}
	req.Header = header
	client := newRetryableHTTPClient()
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting %q: %w", url, err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		msg := fmt.Sprintf("unexpected status code: %d (%s)", response.StatusCode, http.StatusText(response.StatusCode))
		if len(bytes) > 0 {
			msg = fmt.Sprintf("%v: %v", msg, string(bytes))
		}
		return nil, fmt.Errorf(msg)
	}
	return bytes, nil
}

func newRetryableHTTPClient() *http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	return retryClient.StandardClient()
}
