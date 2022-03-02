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

// Package ar implements functions for working with Google Artifact Registry.
package ar

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildermetrics"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"golang.org/x/oauth2/google"
)

const (
	pythonConfigName = ".netrc"
	npmConfigName    = ".npmrc"
)

var (
	npmRegistryRegexp = regexp.MustCompile(`(@[a-zA-Z0-9-]+:)?registry=https:(//[a-zA-Z0-9-]+[-]npm[.]pkg[.]dev/.*/)`)
)

// locations is a list of AR regional endpoints.
var locations = []string{
	"asia",
	"asia-east1",
	"asia-east2",
	"asia-northeast1",
	"asia-northeast2",
	"asia-northeast3",
	"asia-south1",
	"asia-south2",
	"asia-southeast1",
	"asia-southeast2",
	"australia-southeast1",
	"australia-southeast2",
	"europe",
	"europe-central2",
	"europe-north1",
	"europe-west1",
	"europe-west2",
	"europe-west3",
	"europe-west4",
	"europe-west5",
	"europe-west6",
	"northamerica-northeast1",
	"northamerica-northeast2",
	"southamerica-east1",
	"us-central1",
	"us",
	"us-east1",
	"us-east4",
	"us-west1",
	"us-west2",
	"us-west3",
	"us-west4",
}

// arRepositories populates the hosts to be added to the .netrc file.
func arRepositories() []string {
	var arHosts []string
	for _, endpoints := range locations {
		arHosts = append(arHosts, fmt.Sprintf("%s-python.pkg.dev", endpoints))
	}
	return arHosts
}

// GeneratePythonConfig generates a netrc file in the user's HOME directory with the credentials
// necessary for PIP to make authenticated requests to Artifact Registry (see
// https://pip.pypa.io/en/stable/topics/authentication/#netrc-support).
func GeneratePythonConfig(ctx *gcp.Context) error {
	netrcPath := filepath.Join(ctx.HomeDir(), pythonConfigName)
	netrcExists, err := ctx.FileExists(netrcPath)
	if err != nil {
		return err
	}
	if netrcExists {
		ctx.Debugf("Found an existing .netrc file.  Skipping .netrc creation.")
		// If a .netrc file already exists we should not override it.
		return nil
	}

	tok, err := findDefaultCredentials()
	if err != nil {
		// findDefaultCredentials will return an error any time Application Default Credentials are
		// missing (e.g. running the buildpacks locally outside of GCB). Credentials might not
		// be required for the pip install to succeed so we should not fail the build here.
		ctx.Debugf("Unable to find Application Default Credentials. Skipping .netrc creation.")
		return nil
	}

	f, err := ctx.CreateFile(netrcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return writePythonConfig(f, tok)
}

// writePythonConfig writes the .netrc contents for authenticating to AR.
func writePythonConfig(wr io.Writer, tok string) error {
	// pythonConfig is the template for python's .netrc file.
	// A sample config is in token_injector_test.
	const pythonConfig = `
{{- range $entry := .Hosts}}
machine {{$entry}} login oauth2accesstoken password {{$.Token}}
{{- end}}
`
	type authEntry struct {
		Token string
		Hosts []string
	}

	t, err := template.New("netrc").Parse(pythonConfig)
	if err != nil {
		return err
	}

	cfg := authEntry{
		Token: tok,
		Hosts: arRepositories(),
	}

	if err := t.Execute(wr, cfg); err != nil {
		return fmt.Errorf("creating python netrc template: %w", err)
	}

	return nil
}

// GenerateNPMConfig generates an .npmrc file in the user's HOME directory with the credentials
// necessary for NPM to make authenticated requests to Artifact Registry (see
// https://cloud.google.com/artifact-registry/docs/nodejs/authentication).
func GenerateNPMConfig(ctx *gcp.Context) error {
	userConfig := filepath.Join(ctx.HomeDir(), npmConfigName)
	userConfigExists, err := ctx.FileExists(userConfig)
	if err != nil {
		return err
	}
	if userConfigExists {
		ctx.Debugf("Found an existing user-level .npmrc file. Skipping .npmrc creation.")
		return nil
	}

	projectConfig := filepath.Join(ctx.ApplicationRoot(), npmConfigName)
	projConfigExists, err := ctx.FileExists(projectConfig)
	if err != nil {
		return nil
	}
	if !projConfigExists {
		// Unlike Python, NPM credentials must be configured per repo. If the devoloper has not included
		// a project-level npmrc, there are no AR repos to set credentials for, so there is nothing
		// more to do.
		return nil
	}
	content, err := ctx.ReadFile(projectConfig)
	if err != nil {
		return err
	}

	matches := npmRegistryRegexp.FindAllStringSubmatch(string(content), -1)
	var repos []string

	for _, m := range matches {
		repos = append(repos, m[2])
	}

	if len(repos) < 1 {
		return nil
	}

	tok, err := findDefaultCredentials()
	if err != nil {
		// findDefaultCredentials will return an error any time Application Default Credentials are
		// missing (e.g. running the buildpacks locally outside of GCB). Credentials might not
		// be required for the npm install to succeed so we should not fail the build here.
		ctx.Warnf("Skipping .npmrc creation. Unable to find Application Default Credentials: %v", err)
		return nil
	}

	ctx.Debugf("Configuring NPM credentials for: %s", strings.Join(repos, ", "))

	f, err := ctx.CreateFile(userConfig)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeNpmConfig(f, repos, tok)
}

// writeNpmConfig writes the .npmrc contents for authenticating to AR.
func writeNpmConfig(wr io.Writer, repos []string, tok string) error {
	// npmConfig is the template for user level .npmrc that configures repository access tokens.
	const npmConfig = `
{{- range $repo := .Repos}}
{{$repo}}:_authToken={{$.Token}}
{{- end}}
`
	type authEntry struct {
		Token string
		Repos []string
	}

	t, err := template.New("npmrc").Parse(npmConfig)
	if err != nil {
		return err
	}

	cfg := authEntry{
		Token: tok,
		Repos: repos,
	}

	if err := t.Execute(wr, cfg); err != nil {
		return fmt.Errorf("creating NPM .npmrc template: %w", err)
	}

	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.ArNpmCredsGenCounterID).Increment(1)

	return nil
}

// findDefaultCredentials searches for "Application Default Credentials" using the google/oauth
// package (see https://cloud.google.com/docs/authentication/production#automatically).
var findDefaultCredentials = func() (string, error) {
	ctx := context.Background()
	src, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", err
	}
	tok, err := src.TokenSource.Token()
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}
