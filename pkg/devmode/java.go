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

package devmode

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var (
	// JavaWatchedExtensions is the list of file extensions to be watched for changes in Dev Mode for Java.
	// A change to any of those files triggers a rebuild/restart of the application.
	JavaWatchedExtensions = []string{"java", "kt", "scala", "groovy", "clj"}

	// buildScriptTmpl is the template for a build script that runs on each file change in dev mode.
	buildScriptTmpl = template.Must(template.New("script").Parse(`#!/bin/bash
set -e

if [ ! -L {{ .dest }} ]; then
  # The first time the build script runs, it only creates a symlink to the m2 repo.
  # It should skip the build because the application is already built
	ln -s {{ .src }} {{ .dest }}
	exit
fi

# On subsequent runs, it must rebuild the application as the source will have changed.
{{ .buildCommand }}
`))
)

// JavaSyncRules is the list of SyncRules to be configured in Dev Mode for Java.
func JavaSyncRules(dest string) []SyncRule {
	var rules []SyncRule

	for _, ext := range JavaWatchedExtensions {
		rules = append(rules, SyncRule{
			Src:  "**/*." + ext,
			Dest: dest,
		})
	}

	// TODO(dgageot): Also sync resources (html,css,js...).

	return rules
}

// WriteBuildScript writes the build steps to a script to be run on each file change in dev mode.
func WriteBuildScript(ctx *gcp.Context, layerSrc, dest string, command []string) {
	var script bytes.Buffer
	buildScriptTmpl.Execute(&script, map[string]string{
		"src":          layerSrc,
		"dest":         dest,
		"buildCommand": strings.Join(command, " "),
	})

	bin := filepath.Join(layerSrc, "bin")
	ctx.MkdirAll(bin, 0755)
	ctx.WriteFile(filepath.Join(bin, ".devmode_rebuild.sh"), script.Bytes(), 0744)
}
