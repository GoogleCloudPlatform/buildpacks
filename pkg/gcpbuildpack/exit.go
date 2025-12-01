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

package gcpbuildpack

import (
	"os"

	"cloud.google.com/go/logging"
)

// Exiter is responsible to exit the program appropriately; useful for unit tests.
type Exiter interface {
	Exit(exitCode int, err error)
}

type defaultExiter struct {
	ctx *Context
}

func (e defaultExiter) Exit(exitCode int, err error) {
	if err != nil {
		e.ctx.saveErrorOutput(err)
		e.ctx.Logf(divider)

		// Use structured logging if the GOOGLE_ENABLE_STRUCTURED_LOGGING env var is set by RCS.
		if os.Getenv("GOOGLE_ENABLE_STRUCTURED_LOGGING") == "true" {
			e.ctx.StructuredLogf(logging.Error, "%s", err.Error())
		} else {
			e.ctx.Logf("%s", err.Error())
		}

	}

	if exitCode != 0 {
		e.ctx.Tipf(divider)
		e.ctx.Tipf(`Sorry your project couldn't be built.`)
		e.ctx.Tipf(`Our documentation explains ways to configure Buildpacks to better recognise your project:`)
		e.ctx.Tipf(` -> https://cloud.google.com/docs/buildpacks/overview`)
		e.ctx.Tipf(`If you think you've found an issue, please report it:`)
		e.ctx.Tipf(` -> https://github.com/GoogleCloudPlatform/buildpacks/issues/new`)
		e.ctx.Tipf(divider)
	}
	os.Exit(exitCode)
}
