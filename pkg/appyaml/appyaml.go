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

// Package appyaml provides utility methods for working with GAE app.yaml files.
package appyaml

import (
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"gopkg.in/yaml.v2"
)

type appYaml struct {
	Entrypoint    string        `yaml:"entrypoint"`
	RuntimeConfig RuntimeConfig `yaml:"runtime_config"`
}

// RuntimeConfig The runtime_config specified in users app.yaml.
type RuntimeConfig struct {
	DocumentRoot        string `yaml:"document_root"`
	FrontControllerFile string `yaml:"front_controller_file"`
	NginxConfOverride   string `yaml:"nginx_conf_override"`
}

// appYamlIfExists looks up the app.yaml file specified by env var and returns its content if exists.
func appYamlIfExists(root string) (*appYaml, error) {
	exist, path, err := appYamlExists(root)
	if err != nil {
		return nil, err
	}
	if exist {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		a := &appYaml{}
		err = yaml.Unmarshal([]byte(content), &a)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
	return nil, nil
}

// appYamlExists returns true if the specified app.yaml file exists and its path.
func appYamlExists(root string) (bool, string, error) {
	if os.Getenv(env.GaeApplicationYamlPath) == "" {
		return false, "", nil
	}
	path := os.Getenv(env.GaeApplicationYamlPath)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, "", gcp.UserErrorf("Specified app yaml file %v doesn't exist.", path, err)
	}
	if err != nil {
		return false, "", gcp.UserErrorf("Failed to open app yaml file %v.", path, err)
	}
	return true, path, nil
}

// EntrypointIfExists returns entrypoint from GAE app.yaml if it exists.
func EntrypointIfExists(root string) (string, error) {
	a, err := appYamlIfExists(root)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", nil
	}
	if a.Entrypoint == "" {
		return "", gcp.UserErrorf("Couldn't find entrypoint from app.yaml: %s", err)
	}
	return a.Entrypoint, nil
}

// PhpConfiguration returns the PHP configuration in runtime_config
// for GAE Flexible
func PhpConfiguration(root string) (RuntimeConfig, error) {
	a, err := appYamlIfExists(root)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if a == nil {
		return RuntimeConfig{}, nil
	}

	return a.RuntimeConfig, nil
}
