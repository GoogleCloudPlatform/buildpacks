// Copyright 2025 Google LLC
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

// Implements utils/label-image buildpack.
// The label-image buildpack adds any environment variables with the "GOOGLE_LABEL_" prefix as
// labels in the final application image.
package lib

import (
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

// DetectFn is the exported detect function.
func DetectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	return gcp.OptInAlways(), nil
}

// BuildFn is the exported build function.
func BuildFn(ctx *gcp.Context) error {
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, env.LabelPrefix) {
			continue
		}

		parts := strings.SplitN(e, "=", 2)
		key := strings.TrimPrefix(parts[0], env.LabelPrefix)
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		ctx.AddLabel(key, value)
	}
	return nil
}
