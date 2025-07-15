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

	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

var packageRegex = map[string]*regexp.Regexp{
	"gunicorn":  regexp.MustCompile(`(?m)^gunicorn\b([^-]|$)`),
	"uvicorn":   regexp.MustCompile(`(?m)^uvicorn\b([^-]|$)`),
	"gradio":    regexp.MustCompile(`(?m)^gradio\b([^-]|$)`),
	"streamlit": regexp.MustCompile(`(?m)^streamlit\b([^-]|$)`),
}
var eggRegex = map[string]*regexp.Regexp{
	"gunicorn":  regexp.MustCompile(`(?m)#egg=gunicorn$`),
	"uvicorn":   regexp.MustCompile(`(?m)#egg=uvicorn$`),
	"gradio":    regexp.MustCompile(`(?m)#egg=gradio$`),
	"streamlit": regexp.MustCompile(`(?m)#egg=streamlit$`),
}

// PackagePresent checks if a given package is present in the requirements file.
func PackagePresent(ctx *gcpbuildpack.Context, path, name string) (bool, error) {
	requirementsExists, err := ctx.FileExists(path)
	if err != nil || !requirementsExists {
		return false, err
	}
	content, err := ctx.ReadFile(path)
	if err != nil {
		return false, err
	}
	return containsPackage(string(content), name), nil
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
