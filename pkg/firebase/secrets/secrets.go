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

// Package secrets provides functionality around formatting, fetching, and storing secrets in Secret Manager
package secrets

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	secretKeyPrefix = "SECRET_"
	latestSuffix    = "latest"
)

// NormalizeAppHostingSecretsEnv converts the different possible secret formats provided by users
// into one standard format of projects/p/secrets/s/versions/v.
func NormalizeAppHostingSecretsEnv(envMap map[string]string, projectID string) error {
	for k, v := range envMap {
		if strings.HasPrefix(k, secretKeyPrefix) {
			n, err := normalizeSecretFormat(strings.TrimPrefix(k, secretKeyPrefix), v, projectID)
			if err != nil {
				return fmt.Errorf("normalizing secret with key=%v and value=%v: %w", k, v, err)
			}
			envMap[k] = n
		}
	}
	return nil
}

// Handles the following cases:
// "secretID@versionID" -> Extracts the specified secretID and versionID
// "secretID" -> Extracts the specified secretID and uses "latest" for versionID
// "@versionID" -> Uses "envKey" as the secretID and extracts versionID
// "" -> Uses "envKey" as the secretID and "latest" for versionID
func normalizeSecretFormat(envKey, firebaseSecret, projectID string) (string, error) {
	pattern := `^(?P<secretID>\w+)?@?(?P<versionID>\w+)?$`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(firebaseSecret)

	if matches == nil {
		return "", fmt.Errorf("invalid secret format for %v", firebaseSecret)
	}

	secretID := matches[1]
	if secretID == "" {
		secretID = envKey
	}

	versionID := matches[2]
	if versionID == "" {
		versionID = "latest"
	}

	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretID, versionID), nil
}

// PinVersionSecrets will determine the latest version for any secrets that require it and pin it to
// that value for any subsequent steps. Requires that secrets are of the format SECRET_*=projects/p/secrets/s/versions/v
func PinVersionSecrets(envMap map[string]string) error {
	for k, v := range envMap {
		if strings.HasPrefix(k, secretKeyPrefix) && strings.HasSuffix(v, latestSuffix) {
			n, err := getSecretVersion(v)
			if err != nil {
				return fmt.Errorf("calling GetSecretVersion with name=%v: %w", v, err)
			}
			envMap[k] = n
		}
	}
	return nil
}

// DereferenceSecrets will return a mapping of environment variables to their dereferenced secret
// values. Requires that secrets are of the format SECRET_*=projects/p/secrets/s/versions/v
func DereferenceSecrets(envMap map[string]string) (map[string]string, error) {
	dereferencedEnvMap := map[string]string{}
	for k, v := range envMap {
		if strings.HasPrefix(k, secretKeyPrefix) {
			n, err := accessSecretVersion(v)
			if err != nil {
				return nil, fmt.Errorf("calling AccessSecretVersion with name=%v: %w", v, err)
			}
			dereferencedEnvMap[strings.TrimPrefix(k, secretKeyPrefix)] = n
		} else {
			dereferencedEnvMap[k] = v
		}
	}
	return dereferencedEnvMap, nil
}

// Get secret version metadata. Does not include the secret's sensitive data.
func getSecretVersion(name string) (string, error) {
	// TODO (abhisun): Make call to SecretManager GetSecretVersion and extract name field
	return name, nil
}

// Access secret version. Includes secret's sensitive data.
func accessSecretVersion(name string) (string, error) {
	// TODO (abhisun): Make call to SecretManager AccessSecretVersion and extract secret data
	return "secretString", nil
}
