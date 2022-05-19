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

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release/client"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet/release"
	"github.com/Masterminds/semver"
)

func init() {
	acceptance.DefineFlags()
}

// setupTargetFramework modifies the .NET framework version in the project or runtimeconfig file
// for a given test application such that it will specify a value that is associated with the
// acceptance.runtimeVersion parameter.
func setupTargetFramework(setupCtx acceptance.SetupContext) error {
	if setupCtx.RuntimeVersion == "" {
		return nil
	}
	frameworkVersion, err := sdkVersionToFrameworkVersion(setupCtx.RuntimeVersion)
	if err != nil {
		return err
	}
	return filepath.Walk(setupCtx.SrcDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if isProjectFile(path) {
				return setTargetFrameworkInProject(path, frameworkVersion)
			}
			if isRuntimeConfigJSON(path) {
				return applySDKVersionToRuntimeConfigJSON(path, setupCtx.RuntimeVersion, frameworkVersion)
			}
			return nil
		})
}

func isProjectFile(path string) bool {
	return strings.HasSuffix(path, "csproj") || strings.HasSuffix(path, "fsproj") || strings.HasSuffix(path, "vbproj")
}

func isRuntimeConfigJSON(path string) bool {
	return strings.HasSuffix(path, "runtimeconfig.json")
}

func setTargetFrameworkInProject(path, frameworkVersion string) error {
	return replaceStringInFile(
		path,
		"<TargetFramework>.*</TargetFramework>",
		fmt.Sprintf("<TargetFramework>%v</TargetFramework>", frameworkVersion))
}

func replaceStringInFile(path, replRegex, replValue string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file %q: %w", path, err)
	}
	matcher, err := regexp.Compile(replRegex)
	if err != nil {
		return fmt.Errorf("compiling %q to regex: %w", replRegex, err)
	}
	newValue := matcher.ReplaceAllString(string(bytes), replValue)
	if err := os.WriteFile(path, []byte(newValue), 0644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	fmt.Println("TargetFramework: " + replValue)
	return nil
}

func sdkVersionToFrameworkVersion(sdkVersion string) (string, error) {
	semVer, err := semver.NewVersion(sdkVersion)
	if err != nil {
		return "", fmt.Errorf("using %q as semver: %w", sdkVersion, err)
	}
	return fmt.Sprintf("%s%d.%d", getFrameworkPrefix(semVer), semVer.Major(), semVer.Minor()), nil
}

func getFrameworkPrefix(sdkVersion *semver.Version) string {
	net6Ver := semver.MustParse("6.0.0")
	if sdkVersion.LessThan(net6Ver) {
		return "netcoreapp"
	}
	return "net"
}

func applySDKVersionToRuntimeConfigJSON(path, sdkVersion, frameworkVersion string) error {
	dotnetRTVersion, err := release.GetRuntimeVersionForSDKVersion(client.New(), sdkVersion)
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonMap); err != nil {
		return fmt.Errorf("error unmarshalling %q to json: %w", path, err)
	}
	setValueInJSON(jsonMap, frameworkVersion, "runtimeOptions", "tfm")
	setValueInJSON(jsonMap, dotnetRTVersion, "runtimeOptions", "framework", "version")
	bytes, err = json.Marshal(jsonMap)
	if err != nil {
		return fmt.Errorf("marshalling map to json: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}

func setValueInJSON(jsonMap map[string]interface{}, value string, keys ...string) error {
	jsonNode := jsonMap
	if len(keys) > 1 {
		var err error
		jsonNode, err = getNodeInJSON(jsonMap, keys[:len(keys)-1]...)
		if err != nil {
			return err
		}
	}
	lastKey := keys[len(keys)-1]
	jsonNode[lastKey] = value
	return nil
}

func getNodeInJSON(jsonMap map[string]interface{}, keys ...string) (map[string]interface{}, error) {
	curMap := jsonMap
	for idx, k := range keys {
		value, ok := curMap[k]
		if !ok {
			return nil, fmt.Errorf("json missing expected key %q, json value:\n%v", strings.Join(keys[0:idx], "."), jsonMap)
		}
		curMap, ok = value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("value in json at %q is not map[string]interface{} as expected, json value:\n%v",
				k, jsonMap, value)
		}
	}
	return curMap, nil
}
