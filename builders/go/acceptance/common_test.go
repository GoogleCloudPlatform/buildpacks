// Copyright 2022 Google LLC
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

package acceptance_test

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

const (
	goBuild       = "google.go.build"
	goClearSource = "google.go.clear_source"
	goFF          = "google.go.functions-framework"
	goMod         = "google.go.gomod"
	goPath        = "google.go.gopath"
	goRuntime     = "google.go.runtime"
)

func vendorSetup(setupCtx acceptance.SetupContext) error {
	// The setup function runs `go mod vendor` to vendor dependencies
	// specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod vendor",
		setupCtx.SrcDir, setupCtx.Builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vendoring dependencies: %v, output:\n%s", err, out)
	}
	return nil
}

func goSumSetup(setupCtx acceptance.SetupContext) error {
	// The setup function runs `go mod vendor` to vendor dependencies
	// specified in go.mod.
	args := strings.Fields(fmt.Sprintf("docker run --rm -v %s:/workspace -w /workspace -u root %s go mod tidy",
		setupCtx.SrcDir, setupCtx.Builder))
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generating go.sum: %v, output:\n%s", err, out)
	}
	return nil
}
