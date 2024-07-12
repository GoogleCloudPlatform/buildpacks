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
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"regexp"
	"slices"
	"strings"

	"github.com/googleapis/gax-go/v2"

	apphostingschema "github.com/GoogleCloudPlatform/buildpacks/pkg/firebase/apphostingschema"
	smpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// SecretManager is an interface for the Secret Manager API
type SecretManager interface {
	GetSecretVersion(ctx context.Context, req *smpb.GetSecretVersionRequest, opts ...gax.CallOption) (*smpb.SecretVersion, error)
	AccessSecretVersion(ctx context.Context, req *smpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*smpb.AccessSecretVersionResponse, error)
}

var (
	latestSuffix = "latest"
)

// Normalize converts the different possible secret formats provided by users
// into one standard format of projects/p/secrets/s/versions/v.
func Normalize(env []apphostingschema.EnvironmentVariable, projectID string) error {
	for i, ev := range env {
		if ev.Secret != "" {
			n, err := normalizeSecretFormat(ev.Secret, projectID)
			if err != nil {
				return fmt.Errorf("normalizing secret with key=%v and value=%v: %w", ev.Secret, ev.Secret, err)
			}
			env[i].Secret = n
		}
	}

	return nil
}

var (
	patternBare          = regexp.MustCompile(`^[^/@]+$`)
	patternBareVersioned = regexp.MustCompile(`^([^/@]+)@([0-9]+)$`)
	patternFull          = regexp.MustCompile(`^projects/([^/]+)/secrets/([^/]+)$`)
	patternFullVersioned = regexp.MustCompile(`^projects/([^/]+)/secrets/([^/]+)/versions/([^/]+)$`)
)

// Handles the following cases:
// 1. "secretID" -> Extracts the specified secretID and uses "latest" for versionID
// 2. "secretID@versionID" -> Extracts the specified secretID and versionID
// 3. "projects/projectID/secrets/secretID" -> Uses "latest" for versionID
// 4. "projects/projectID/secrets/secretID/versions/versionID" -> Uses as is
func normalizeSecretFormat(firebaseSecret, projectID string) (string, error) {
	// Handle "secretID"
	if patternBare.MatchString(firebaseSecret) {
		return fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, firebaseSecret), nil
	}

	// Handle "secretID@versionID"
	if patternBareVersioned.MatchString(firebaseSecret) {
		matches := patternBareVersioned.FindStringSubmatch(firebaseSecret)
		return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, matches[1], matches[2]), nil
	}

	// Handle "projects/projectID/secrets/secretID"
	if patternFull.MatchString(firebaseSecret) {
		return firebaseSecret + "/versions/latest", nil
	}

	// Handle "projects/projectID/secrets/secretID/versions/versionID"
	if patternFullVersioned.MatchString(firebaseSecret) {
		return firebaseSecret, nil
	}

	return "", fmt.Errorf("invalid secret format for %v", firebaseSecret)
}

// PinVersions will determine the latest version for any secrets that require it and pin it to
// that value for any subsequent steps. Requires that secrets are of the format 'projects/p/secrets/s/versions/v'
func PinVersions(ctx context.Context, client SecretManager, env []apphostingschema.EnvironmentVariable) error {
	for i, ev := range env {
		if ev.Secret != "" && strings.HasSuffix(ev.Secret, latestSuffix) {
			n, err := getSecretVersion(ctx, client, ev.Secret)
			if err != nil {
				return fmt.Errorf(
					"calling GetSecretVersion with name=%v: %w. "+
						"If the secret already exists in your project, please grant your App Hosting backend "+
						"access to it with the CLI command 'firebase apphosting:secrets:grantaccess'. "+
						"See https://firebase.google.com/docs/app-hosting/configure#secret-parameters for more information",
					ev.Secret,
					err,
				)
			}
			env[i].Secret = n
		}
	}

	return nil
}

// GenerateBuildDereferencedEnvMap will return a mapping of environment variables to their dereferenced
// secret values along with plain env vars only if they are scope to BUILD availability. Requires
// that secrets are of the format 'projects/p/secrets/s/versions/v'
func GenerateBuildDereferencedEnvMap(ctx context.Context, client SecretManager, env []apphostingschema.EnvironmentVariable) (map[string]string, error) {
	dereferencedEnvMap := map[string]string{}

	for _, ev := range env {
		if slices.Contains(ev.Availability, "BUILD") {
			if ev.Value != "" {
				dereferencedEnvMap[ev.Variable] = ev.Value
			} else if ev.Secret != "" {
				n, err := accessSecretVersion(ctx, client, ev.Secret)
				if err != nil {
					return nil, fmt.Errorf("calling AccessSecretVersion with name=%v: %w", ev.Secret, err)
				}
				dereferencedEnvMap[ev.Variable] = n
			}
		}
	}

	return dereferencedEnvMap, nil
}

// Get secret version metadata. Does NOT include the secret's sensitive data.
func getSecretVersion(ctx context.Context, client SecretManager, name string) (string, error) {
	req := &smpb.GetSecretVersionRequest{
		Name: name,
	}

	result, err := client.GetSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("getting secret version: %w", err)
	}
	log.Printf("Pinned secret %v to %v for the rest of the current build and run", name, result.Name)
	return result.Name, nil
}

// Access secret version. Includes secret's sensitive data.
func accessSecretVersion(ctx context.Context, client SecretManager, name string) (string, error) {
	result, err := client.AccessSecretVersion(ctx, &smpb.AccessSecretVersionRequest{
		Name: name,
	})

	if err != nil {
		return "", fmt.Errorf("failed to access secret version %v: %w", name, err)
	}
	log.Printf("Accessed secret %v for the rest of the current build", name)

	// Verify the data checksum.
	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))

	if checksum != *result.Payload.DataCrc32C {
		return "", fmt.Errorf("data corruption while accessing secret %v detected", name)
	}

	return string(result.Payload.Data), nil
}
