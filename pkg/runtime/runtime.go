// Copyright 2020 Google LLC
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

// Package runtime is used to perform general runtime actions.
package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// CheckOverride checks GOOGLE_RUNTIME and opts in or opts out as appropriate. If GOOGLE_RUNTIME is not set, or invalid, no action is taken.
// The functions returns a boolean indicating whether the detect function should exit.
func CheckOverride(ctx *gcp.Context, wantRuntime string) gcp.DetectResult {
	er := strings.ToLower(strings.TrimSpace(os.Getenv(env.Runtime)))
	if er == "" {
		return nil
	}

	if er != wantRuntime {
		return gcp.OptOut(fmt.Sprintf("%s not set to %q", env.Runtime, wantRuntime))
	}
	return gcp.OptIn(fmt.Sprintf("%s set to %q", env.Runtime, wantRuntime))
}
