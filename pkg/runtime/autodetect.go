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

package runtime

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/autodetect"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/version"
	"github.com/buildpacks/libcnb/v2"
	"github.com/Masterminds/semver"
)

// AutodetectedRuntime is a runtime that was selected for automatic base image updates.
type AutodetectedRuntime = autodetect.Runtime

// SelectAutodetectRuntime returns the runtime of the given language (e.g. "nodejs") that is eligible
// for automatic base image updates for the requested version constraint. The eligible runtimes
// and their versions come from the X_GOOGLE_RUNTIME_VERSIONS env var, a JSON object that maps
// runtime names to versions.
//
// The runtime is selected as follows:
//   - If the version of one or more eligible runtimes satisfies the constraint, the one with the
//     highest version is returned. Constraints are matched with the same rules that are used to
//     resolve the version to install, so detect and build agree on the version.
//   - Otherwise, if the constraint pins a version (e.g. "26.1.0", "26.1" or "3.14.9"), the eligible
//     runtime of the same release line is returned, e.g. nodejs26 for "26.1.0" or python314 for
//     "3.14.9". The app then builds and runs on that runtime's version instead of the pinned one.
//
// It returns nil, nil if X_GOOGLE_RUNTIME_VERSIONS is not set or no eligible runtime matches. It
// returns an error if X_GOOGLE_RUNTIME_VERSIONS isn't a JSON object of versions or the constraint
// can't be parsed, so that neither looks like an app whose version matches no eligible runtime.
func SelectAutodetectRuntime(language, constraint string) (*AutodetectedRuntime, error) {
	eligible, err := autodetectRuntimes()
	if err != nil || len(eligible) == 0 {
		return nil, err
	}
	nameRegexp := regexp.MustCompile("^" + regexp.QuoteMeta(language) + "([0-9]+)$")
	byVersion := map[string]string{}
	var versions []string
	for name, v := range eligible {
		m := nameRegexp.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		// Skip placeholders, whose version is of another release line, e.g. php86 at 8.5.12.
		if _, ok := releaseLine(v, m[1]); !ok {
			continue
		}
		if other, ok := byVersion[v]; ok {
			return nil, fmt.Errorf("runtimes %q and %q have the same version %q in %s", other, name, v, env.XGoogleRuntimeVersions)
		}
		byVersion[v] = name
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return nil, nil
	}
	if constraint == "" {
		constraint = "*"
	}
	if _, err := semver.NewConstraint(constraint); err != nil {
		return nil, fmt.Errorf("parsing the requested %s version %q: %w", language, constraint, err)
	}
	if v, err := version.ResolveVersion(constraint, versions, version.WithoutSanitization); err == nil {
		return &AutodetectedRuntime{Name: byVersion[v], Version: v}, nil
	}
	// No eligible version satisfies the constraint; fall back to the release line of a pinned version.
	for v, name := range byVersion {
		suffix := nameRegexp.FindStringSubmatch(name)[1]
		if line, ok := releaseLine(v, suffix); ok && pinsReleaseLine(constraint, line) {
			return &AutodetectedRuntime{Name: name, Version: v}, nil
		}
	}
	return nil, nil
}

// pinnedVersionRegexp matches a plain version without operators, e.g. "26", "26.1", "3.14.9" or
// "v26.1.x".
var pinnedVersionRegexp = regexp.MustCompile(`^v?[0-9]+(\.([0-9]+|x|X|\*))*$`)

// releaseLine returns the leading components of version that identify the runtime's release line,
// i.e. the components whose concatenation is the runtime name suffix. For example the release line
// of nodejs24 (24.22.1) is "24", of python314 (3.14.10) "3.14" and of go127 (1.27.3) "1.27".
func releaseLine(version, nameSuffix string) (string, bool) {
	parts := strings.Split(version, ".")
	for n := 1; n <= len(parts); n++ {
		if strings.Join(parts[:n], "") == nameSuffix {
			return strings.Join(parts[:n], "."), true
		}
	}
	return "", false
}

// pinsReleaseLine reports whether constraint is a plain version that pins a version within line,
// e.g. "26.1.0" or "26.1" for line "26", and "3.14.9" for line "3.14". Ranges such as "~26.1.0"
// and versions that don't specify the whole release line (e.g. "3" for line "3.14") don't match.
func pinsReleaseLine(constraint, line string) bool {
	c := strings.TrimSpace(constraint)
	if !pinnedVersionRegexp.MatchString(c) {
		return false
	}
	c = strings.TrimPrefix(c, "v")
	return c == line || strings.HasPrefix(c, line+".")
}

// AutodetectPlan returns a build plan that records rt for automatic base image updates. The
// buildpack both provides and requires the entry so that it never affects which group passes
// detection.
func AutodetectPlan(rt *AutodetectedRuntime) libcnb.BuildPlan {
	return libcnb.BuildPlan{
		Provides: []libcnb.BuildPlanProvide{{Name: autodetect.PlanName}},
		Requires: []libcnb.BuildPlanRequire{{
			Name: autodetect.PlanName,
			Metadata: map[string]any{
				autodetect.PlanRuntimeKey: rt.Name,
				autodetect.PlanVersionKey: rt.Version,
			},
		}},
	}
}

// WithAutodetectPlan returns a detect result option that records the runtime of the given language
// when the runtime is detected automatically, i.e. when X_GOOGLE_RUNTIME_VERSIONS is set.
// constraint returns the runtime version constraint requested by the application.
//
// The option is evaluated lazily: constraint is only called when the option is applied and
// X_GOOGLE_RUNTIME_VERSIONS is set, so pass the option only to opt-in results. Automatic runtime
// detection must never fail detection, so errors are logged and no build plan is added.
func WithAutodetectPlan(ctx *gcp.Context, language string, constraint func() (string, error)) gcp.DetectResultOption {
	return gcp.WithBuildPlansFunc(func() []libcnb.BuildPlan {
		if os.Getenv(env.XGoogleRuntimeVersions) == "" {
			return nil
		}
		c, err := constraint()
		if err != nil {
			ctx.Warnf("Automatic runtime detection: determining the requested %s version: %v", language, err)
			return nil
		}
		rt, err := SelectAutodetectRuntime(language, c)
		if err != nil {
			ctx.Warnf("Automatic runtime detection: %v", err)
			return nil
		}
		if rt == nil {
			ctx.Logf("Automatic runtime detection: no eligible %s runtime satisfies version %q", language, c)
			return nil
		}
		ctx.Logf("Automatic runtime detection: selected %s (version %s)", rt.Name, rt.Version)
		return []libcnb.BuildPlan{AutodetectPlan(rt)}
	})
}

// autodetectRuntimes parses X_GOOGLE_RUNTIME_VERSIONS with autodetect.ParseRuntimeVersions.
func autodetectRuntimes() (map[string]string, error) {
	runtimes, err := autodetect.ParseRuntimeVersions(os.Getenv(env.XGoogleRuntimeVersions))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", env.XGoogleRuntimeVersions, err)
	}
	return runtimes, nil
}
