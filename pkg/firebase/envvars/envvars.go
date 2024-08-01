// Copyright 2024 Google LLC
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

// Package envvars handles the writing of .env-esque files to disk for use in subsequent
// Cloud Build steps.
package envvars

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
)

// Write produces a file where each like has the format KEY=VALUE. We aren't using the
// godotenv library as its output isn't compatible with the `pack build --env-file` command.
func Write(env map[string]string, fileName string) error {
	content, err := marshal(env)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, []byte(content+"\n"), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Read reads in the custom env file to a map. This is a very dumb function that
// just splits each line with the format KEY=VALUE and adds it to the output map.
func Read(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if equalSignIndex := strings.Index(line, "="); equalSignIndex != -1 {
			key := line[:equalSignIndex]
			value := line[equalSignIndex+1:]
			envMap[key] = value
		} else if line != "" {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envMap, nil
}

func marshal(envMap map[string]string) (string, error) {
	var lines []string
	for k, v := range envMap {
		if d, err := strconv.Atoi(v); err == nil {
			lines = append(lines, fmt.Sprintf(`%s=%d`, k, d))
		} else {
			// String replacement is needed to properly escape the newline character
			lines = append(lines, fmt.Sprintf(`%s=%s`, k, strings.ReplaceAll(v, "\n", "\\n")))
		}
	}
	// Sorting as the iteration order of a map is not guaranteed to be the same every time.
	// Needed for some test assertions.
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

// ParseEnvVarsFromString parses the server side environment variables from a string to a list of EnvironmentVariables.
func ParseEnvVarsFromString(serverSideEnvVars string) ([]apphostingschema.EnvironmentVariable, error) {
	var parsedServerSideEnvVars []apphostingschema.EnvironmentVariable

	err := json.Unmarshal([]byte(serverSideEnvVars), &parsedServerSideEnvVars)
	if err != nil {
		return parsedServerSideEnvVars, fmt.Errorf("unmarshalling server side env var %v: %w", serverSideEnvVars, err)
	}

	return parsedServerSideEnvVars, nil
}
