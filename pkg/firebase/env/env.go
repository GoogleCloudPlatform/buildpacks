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

// Package env provides functionality around reading, processing, and writing environment variable
// files.
package env

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var (
	reservedKeys = map[string]bool{
		"PORT":            true,
		"K_SERVICE":       true,
		"K_REVISION":      true,
		"K_CONFIGURATION": true,
	}

	reservedFirebaseKeyPrefix = "FIREBASE_"
	secretKeyPrefix           = "SECRET_"
)

// ReadEnv parses environment variables at the given file path.
func ReadEnv(path string) (map[string]string, error) {
	envMap, err := godotenv.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Missing environment variables at path %v, skipping", path)
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading environment variables at %v: %w", path, err)
	}
	return envMap, nil
}

// WriteEnv writes environment variables to the given file path.
func WriteEnv(envMap map[string]string, path string) error {
	return godotenv.Write(envMap, path)
}

func isReservedKey(envKey string) bool {
	if _, ok := reservedKeys[envKey]; ok {
		return true
	} else if strings.HasPrefix(envKey, reservedFirebaseKeyPrefix) {
		return true
	}
	return false
}

// SanitizeAppHostingEnv strips reserved environment variables from an environment variable map.
func SanitizeAppHostingEnv(envMap map[string]string) (map[string]string, error) {
	sanitizedEnvMap := map[string]string{}
	for k, v := range envMap {
		if !isReservedKey(k) {
			sanitizedEnvMap[k] = v
		} else {
			log.Printf("WARNING: %s is a reserved key, removing it from the final environment variables", k)
		}
	}
	return sanitizedEnvMap, nil
}
