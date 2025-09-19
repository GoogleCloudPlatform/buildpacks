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

package python

import (
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/BurntSushi/toml"
)

var (
	requirements = "requirements.txt"
)

var packageRegex = map[string]*regexp.Regexp{
	"gunicorn":            regexp.MustCompile(`(?m)^gunicorn\b([^-]|$)`),
	"uvicorn":             regexp.MustCompile(`(?m)^uvicorn\b([^-]|$)`),
	"gradio":              regexp.MustCompile(`(?m)^gradio\b([^-]|$)`),
	"streamlit":           regexp.MustCompile(`(?m)^streamlit\b([^-]|$)`),
	"fastapi[standard]":   regexp.MustCompile(`(?m)^fastapi\[standard\]([^-]|$)`),
	"functions-framework": regexp.MustCompile(`(?m)^functions-framework\b([^-]|$)`),
}
var eggRegex = map[string]*regexp.Regexp{
	"gunicorn":            regexp.MustCompile(`(?m)#egg=gunicorn$`),
	"uvicorn":             regexp.MustCompile(`(?m)#egg=uvicorn$`),
	"gradio":              regexp.MustCompile(`(?m)#egg=gradio$`),
	"streamlit":           regexp.MustCompile(`(?m)#egg=streamlit$`),
	"fastapi[standard]":   regexp.MustCompile(`(?m)#egg=fastapi\[standard\]$`),
	"functions-framework": regexp.MustCompile(`(?m)#egg=functions-framework$`),
}

// PackagePresent checks if a given package is present in the requirements file.
func PackagePresent(ctx *gcpbuildpack.Context, name string) (bool, error) {
	requirementsExists, err := ctx.FileExists(requirements)
	if err != nil {
		return false, err
	}
	if requirementsExists {
		return RequirementsPackagePresent(ctx, name)
	}
	pyprojectTomlExists, err := ctx.FileExists(pyprojectToml)
	if err != nil || !pyprojectTomlExists {
		return false, err
	}
	if pyprojectTomlExists && IsPyprojectEnabled() {
		return PyprojectPackagePresent(ctx, name)
	}
	return false, nil
}

// RequirementsPackagePresent checks if a given package is present in requirements.txt.
func RequirementsPackagePresent(ctx *gcpbuildpack.Context, name string) (bool, error) {
	content, err := ctx.ReadFile(requirements)
	if err != nil {
		return false, err
	}
	return containsPackage(string(content), name), nil
}

// PyprojectPackagePresent checks if a given package is present in pyproject.toml.
func PyprojectPackagePresent(ctx *gcpbuildpack.Context, name string) (bool, error) {
	content, err := ctx.ReadFile(pyprojectToml)
	if err != nil {
		return false, err
	}

	var parsedTOML struct {
		Project struct {
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Dependencies map[string]any `toml:"dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}

	if _, err := toml.Decode(string(content), &parsedTOML); err != nil {
		ctx.Warnf("Could not parse %s: %v", pyprojectToml, err)
		return false, err
	}

	if containsPackage(strings.Join(parsedTOML.Project.Dependencies, "\n"), name) {
		return true, nil
	}

	if _, exists := parsedTOML.Tool.Poetry.Dependencies[name]; exists {
		return true, nil
	}

	return false, nil
}

func containsPackage(s, name string) bool {
	re, ok := packageRegex[name]
	if !ok {
		return false // Or handle error
	}
	eggRe, ok := eggRegex[name]
	if !ok {
		return false // Or handle error
	}
	return re.MatchString(s) || eggRe.MatchString(s)
}
