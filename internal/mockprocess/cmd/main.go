// Package main is a mock process thats behavior can be configured by
// by setting certain environment variables.
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/internal/mockprocess/mockprocessutil"
)

func main() {
	mocksJSON := os.Getenv(mockprocessutil.EnvHelperMockProcessMap)
	if mocksJSON == "" {
		log.Fatalf("%q env var must be set", mockprocessutil.EnvHelperMockProcessMap)
	}
	mockProcesses, err := mockprocessutil.UnmarshalMockProcessMap(mocksJSON)
	if err != nil {
		log.Fatalf("unable to unmarshal mock process map from JSON '%s': %v", mocksJSON, err)
	}

	fullCommand := strings.Join(os.Args[1:], " ")
	var mockMatch *mockprocessutil.MockProcessConfig = nil
	for commandRegex, mock := range mockProcesses {
		re := regexp.MustCompile(commandRegex)
		if re.MatchString(fullCommand) {
			mockMatch = mock
			break
		}
	}
	if mockMatch == nil {
		// To avoid needing to mock every call to Exec, assume
		// the process should pass if it wasn't specified by the test.
		os.Exit(0)
	}

	if mockMatch.Stdout != "" {
		fmt.Fprint(os.Stdout, mockMatch.Stdout)
	}

	if mockMatch.Stderr != "" {
		fmt.Fprint(os.Stderr, mockMatch.Stderr)
	}

	os.Exit(mockMatch.ExitCode)
}
