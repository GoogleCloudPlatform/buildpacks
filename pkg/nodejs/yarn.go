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

package nodejs

import (
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

const (
	// YarnLock is the name of the yarn lock file.
	YarnLock = "yarn.lock"
)

// LockfileFlag returns an appropriate lockfile handling flag, including empty string.
func LockfileFlag(ctx *gcp.Context) (string, error) {
	// HACK: For backwards compatibility on App Engine Node.js 10 and older, skip using `--frozen-lockfile`.
	if isOldNode, err := isPreNode11(ctx); err != nil || isOldNode {
		return "", err
	}
	return "--frozen-lockfile", nil
}
