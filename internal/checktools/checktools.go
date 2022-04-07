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

// Package checktools provides functionns to check all tools are correctly installed.
package checktools

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver"
)

var (
	// LINT.IfChange(minPackVersion)
	// minPackVersion is the minimum required version of pack.
	minPackVersion = semver.MustParse("0.23.0")
	// LINT.ThenChange(//depot/google3/apphosting/g3doc/runtimes/tutorials/buildpack-tests-debug.md)
)

// Installed checks that all required tools are on PATH.
func Installed() error {
	tools := []struct {
		name string
		url  string
	}{
		{"pack", "https://buildpacks.io/docs/install-pack/"},
		{"docker", "https://docs.docker.com/install/"},
		{"container-structure-test", "https://github.com/GoogleContainerTools/container-structure-test#installation"},
	}

	for _, tool := range tools {
		path, err := exec.LookPath(tool.name)
		if err != nil {
			return fmt.Errorf("%s not found, please follow %s and ensure it is on $PATH: %s", tool.name, tool.url, os.Getenv("PATH"))
		}
		log.Printf("%s found at %s", tool.name, path)
	}
	return nil
}

// PackVersion checks that the installed pack has the correct version.
func PackVersion() error {
	// pack requires $HOME to exist.
	home, err := ioutil.TempDir("", "pack-home")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)

	path, err := exec.LookPath("pack")
	if err != nil {
		return fmt.Errorf("finding pack on PATH: %w", err)
	}
	cmd := exec.Command(path, "--version")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running pack: %v", err)
	}
	log.Printf("Found pack version %s", out)
	// Remove the build suffix so it can be parsed by semver andt the error message is less confusing.
	// For example, instead of `0.13.1+git-4134cc6.build-1135` use `0.13.1`.
	version, err := semver.NewVersion(strings.Split(string(out), "+")[0])
	if err != nil {
		return fmt.Errorf("parsing semver from %s: %v", out, err)
	}

	if version.LessThan(minPackVersion) {
		return fmt.Errorf("outdated pack binary (%s): want %s, got %s; to update please follow https://buildpacks.io/docs/install-pack/", path, minPackVersion, version)
	}
	return nil
}
