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

// Implements utils/archive-source buildpack.
// The archive-source buildpack archives user's source code.
package main

import (
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) error {
	return nil
}

func buildFn(ctx *gcp.Context) error {
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
