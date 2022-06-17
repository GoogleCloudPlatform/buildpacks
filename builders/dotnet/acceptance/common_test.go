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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
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
			return nil
		})
}

func isProjectFile(path string) bool {
	return strings.HasSuffix(path, "csproj") || strings.HasSuffix(path, "fsproj") || strings.HasSuffix(path, "vbproj")
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
