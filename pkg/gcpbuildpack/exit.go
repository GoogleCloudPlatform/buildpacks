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
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/buildererror"
)

// Exiter is responsible to exit the program appropriately; useful for unit tests.
type Exiter interface {
	Exit(exitCode int, be *buildererror.Error)
}

type defaultExiter struct {
	ctx *Context
}

func (e defaultExiter) Exit(exitCode int, be *buildererror.Error) {
	if be != nil {
		msg := "Failure: "
		if be.ID != "" {
			msg += fmt.Sprintf("(ID: %s) ", be.ID)
		}
		msg += be.Message
		e.ctx.Logf(msg)
		e.ctx.saveErrorOutput(be)
	}

	if exitCode != 0 {
		e.ctx.Tipf(divider)
		e.ctx.Tipf(`Sorry your project couldn't be built.`)
		e.ctx.Tipf(`Our documentation explains ways to configure Buildpacks to better recognise your project:`)
		e.ctx.Tipf(` -> https://github.com/GoogleCloudPlatform/buildpacks/blob/main/README.md`)
		e.ctx.Tipf(`If you think you've found an issue, please report it:`)
		e.ctx.Tipf(` -> https://github.com/GoogleCloudPlatform/buildpacks/issues/new`)
		e.ctx.Tipf(divider)
	}

	os.Exit(exitCode)
}
