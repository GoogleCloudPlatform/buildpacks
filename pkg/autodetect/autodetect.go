// Copyright 2026 Google LLC
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

// Package autodetect has the parts of automatic runtime detection that both sides share: the
// runtime buildpacks, which record the detected runtime in the build plan, and the platform, which
// reads it from the build plan to build on that runtime's base image.
package autodetect

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

const (
	// PlanName is the name of the build plan entry that records the detected runtime.
	PlanName = "google.runtime"
	// PlanRuntimeKey is the build plan metadata key for the runtime name, e.g. "nodejs24".
	PlanRuntimeKey = "runtime"
	// PlanVersionKey is the build plan metadata key for the exact runtime version.
	PlanVersionKey = "version"
)

// Runtime is a detected runtime.
type Runtime struct {
	// Name is the runtime name, e.g. "nodejs24".
	Name string
	// Version is the exact runtime version, e.g. "24.22.1".
	Version string
}

// ParseRuntimeVersions parses the value of X_GOOGLE_RUNTIME_VERSIONS, a JSON object that maps the
// runtimes that may be detected to their versions, e.g. {"nodejs24":"24.22.1"}. It returns nil if
// raw is empty.
//
// The platform generates the value, so an invalid entry is a platform bug. Rather than letting it
// look like an app whose version matches no runtime, the whole value is rejected with an error
// that names the entry. A version must be a semantic version that version.ResolveVersion does not
// skip, i.e. not a release candidate.
func ParseRuntimeVersions(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var runtimes map[string]string
	if err := json.Unmarshal([]byte(raw), &runtimes); err != nil {
		return nil, fmt.Errorf("parsing %q as a JSON object of runtime versions: %w", raw, err)
	}
	for name, v := range runtimes {
		if name == "" || v == "" {
			return nil, fmt.Errorf("invalid runtime %q with version %q", name, v)
		}
		if _, err := semver.NewVersion(v); err != nil {
			return nil, fmt.Errorf("invalid version %q of runtime %q: %w", v, name, err)
		}
		if version.IsReleaseCandidate(v) {
			return nil, fmt.Errorf("invalid version %q of runtime %q: release candidates are not eligible", v, name)
		}
	}
	return runtimes, nil
}

// planTOML is the part of the lifecycle's plan.toml that has the build plan
// entries. Their requirements are libcnb.BuildPlanRequire, as buildpacks write
// them.
type planTOML struct {
	Entries []struct {
		Requires []libcnb.BuildPlanRequire `toml:"requires"`
	} `toml:"entries"`
}

// RuntimeFromPlan returns the runtime recorded in plan, the contents of the lifecycle's plan.toml,
// or nil if none is. runtimes are the runtimes that may be detected, mapped to their versions.
// Build plan entries other than PlanName are ignored.
//
// It returns an error if the recorded runtime is not one of runtimes at its version, or if more
// than one runtime is recorded, since the build can only use one.
func RuntimeFromPlan(plan []byte, runtimes map[string]string) (*Runtime, error) {
	var p planTOML
	if err := toml.Unmarshal(plan, &p); err != nil {
		return nil, fmt.Errorf("parsing the build plan: %w", err)
	}
	var detected *Runtime
	for _, entry := range p.Entries {
		for _, require := range entry.Requires {
			if require.Name != PlanName {
				continue
			}
			name, _ := require.Metadata[PlanRuntimeKey].(string)
			v, _ := require.Metadata[PlanVersionKey].(string)
			if want, ok := runtimes[name]; !ok || v != want {
				return nil, fmt.Errorf("%q build plan entry records runtime %q version %q, which is not one of the offered runtimes", PlanName, name, v)
			}
			// Entries that record the same runtime are fine; different ones are ambiguous.
			if detected != nil && detected.Name != name {
				return nil, fmt.Errorf("%q build plan entries record both %s and %s", PlanName, detected.Name, name)
			}
			detected = &Runtime{Name: name, Version: v}
		}
	}
	return detected, nil
}
