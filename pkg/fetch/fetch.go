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

// Package fetch contains functions for downloading various content types via HTTP.
package fetch

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/hashicorp/go-retryablehttp"
)

// gcpUserAgent is required for the Ruby runtime, but used for others for simplicity.
const gcpUserAgent = "GCPBuildpacks"

// Tarball downloads a tarball from a URL and extracts it into the provided directory.
func Tarball(url, dir string, stripComponents int) error {
	response, err := doGet(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return untar(dir, response.Body, stripComponents)
}

// ARImage downloads tarball from images in artifact registry.
func ARImage(url, dir string, stripComponents int) error {
	image, err := crane.Pull(url)
	if err != nil {
		return err
	}
	layers, err := image.Layers()
	if err != nil {
		return err
	}
	if len(layers) < 1 {
		return gcp.InternalErrorf("runtime image has no layer")
	}
	l := layers[0]
	rc, err := l.Compressed()
	if err != nil {
		return err
	}
	defer rc.Close()
	return untar(dir, rc, stripComponents)
}

// File downloads a file from a URL and writes it to the provided path.
func File(url, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	response, err := doGet(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, err = io.Copy(out, response.Body)
	return err
}

// JSON fetches a JSON payload from a URL and unmarshals it into the value pointed to by v.
func JSON(url string, v interface{}) error {
	response, err := doGet(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return gcp.InternalErrorf("reading response body from %q: %v", url, err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return gcp.InternalErrorf("decoding response from %q: %v", url, err)
	}
	return nil
}

// GetURL makes an HTTP GET request to given URL and writes the body to the provided writer.
func GetURL(url string, f io.Writer) error {
	response, err := doGet(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if _, err = io.Copy(f, response.Body); err != nil {
		return gcp.InternalErrorf("copying response body: %v", err)
	}

	return nil
}

// untar extracts a tarball from a reader and writes it to the given directory.
func untar(dir string, r io.Reader, stripComponents int) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return gcp.InternalErrorf("creating gzip reader: %v", err)
	}
	defer gzr.Close()

	madeDir := map[string]bool{}
	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return gcp.InternalErrorf("untaring file: %v", err)
		case header == nil:
			continue
		}

		target, err := tarDestination(header.Name, dir, header.Typeflag, stripComponents)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.Mkdir(target, os.FileMode(header.Mode)); err != nil {
					return gcp.InternalErrorf("creating directory %q: %v", target, err)
				}
				madeDir[target] = true
			}
		case tar.TypeReg, tar.TypeRegA:
			// Make the directory. This is redundant because it should
			// already be made by a directory entry in the tar
			// beforehand. Thus, don't check for errors; the next
			// write will fail with the same error.
			dir := filepath.Dir(target)
			if !madeDir[dir] {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return gcp.InternalErrorf("creating directory %q: %v", target, err)
				}
				madeDir[dir] = true
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return gcp.InternalErrorf("opening file %q: %v", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				return gcp.InternalErrorf("copying file %q: %v", target, err)
			}
			if err := f.Close(); err != nil {
				return gcp.InternalErrorf("closing file %q: %v", target, err)
			}
		case tar.TypeSymlink:
			targetPath := filepath.Join(filepath.Dir(target), header.Linkname)
			if !isValidTarDestination(targetPath, dir, header.Typeflag) {
				return gcp.InternalErrorf("symlink %q -> %q traverses out of root", target, header.Linkname)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return gcp.InternalErrorf("symlinking %q to %q: %v", target, header.Linkname, err)
			}
		case tar.TypeLink:
			link, err := tarDestination(header.Linkname, dir, header.Typeflag, stripComponents)
			if err != nil {
				return err
			}
			if err := os.Link(link, target); err != nil {
				return gcp.InternalErrorf("linking %q to %q: %v", target, link, err)
			}
		default:
			return gcp.InternalErrorf("invalid tar entry %v", header)
		}
	}
}

// tarDestination returns the filepath that a tar entry should be written to when extracted.
func tarDestination(tarPath, rootDir string, tarType byte, stripComponents int) (string, error) {
	rootDir = filepath.Clean(rootDir)
	path := filepath.Join(rootDir, filepath.Clean(tarPath))

	if stripComponents > 0 {
		drop := strings.Count(rootDir, string(filepath.Separator)) + stripComponents + 1
		parts := strings.Split(path, string(filepath.Separator))
		if drop >= len(parts) && tarType == tar.TypeDir {
			// This is a stripped away directory, returning rootDir makes this a no-op.
			return rootDir, nil
		}
		if drop >= len(parts) {
			// This is a file that would have been dropped if stripped it.
			return "", gcp.InternalErrorf("stripped too many components (%v)", stripComponents)
		}
		path = filepath.Join(rootDir, filepath.Join(parts[drop:]...))
	}

	// Only allow extraction either directly into the root, or within a subdirectory from the root.
	if isValidTarDestination(path, rootDir, tarType) {
		return path, nil
	}
	return "", gcp.InternalErrorf("tar entry %q traverses out of root", tarPath)
}

// isValidTarDestination protects against a path traversal vulnerability by ensuring the final path
// is within the target directory.
func isValidTarDestination(dest, rootDir string, tarType byte) bool {
	destDir := dest
	if tarType != tar.TypeDir {
		destDir = filepath.Dir(dest)
	}
	return destDir == rootDir ||
		strings.HasPrefix(destDir, rootDir+string(filepath.Separator))
}

// doGet performs an HTTP GET request for a URL.
func doGet(url string) (*http.Response, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, gcp.UserErrorf("fetching %s: %v", url, err)
	}

	req.Header.Set("User-Agent", gcpUserAgent)

	response, err := retryClient.StandardClient().Do(req)
	if err != nil {
		return nil, gcp.UserErrorf("requesting %s: %v", url, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		defer response.Body.Close()
		return nil, gcp.UserErrorf("fetching %s returned HTTP status: %d", url, response.StatusCode)
	}
	return response, err
}
