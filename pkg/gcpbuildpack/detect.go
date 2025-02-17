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

	"github.com/buildpacks/libcnb/v2"
)

// DetectResult represents the result of the detect run and the reason for it.
type DetectResult interface {
	Result() libcnb.DetectResult
	Reason() string
}

type detectResult struct {
	result libcnb.DetectResult
	reason string
}

func (d *detectResult) Result() libcnb.DetectResult {
	return d.result
}

func (d *detectResult) Reason() string {
	return d.reason
}

// DetectResultOption is a function that returns strings to be hashed when computing a cache key.
type DetectResultOption func(r *detectResult)

// WithBuildPlans adds build plans to the detect result.
func WithBuildPlans(plans ...libcnb.BuildPlan) DetectResultOption {
	return func(r *detectResult) {
		r.result.Plans = plans
	}
}

// OptIn is used during the detect phase to opt in to the build process.
func OptIn(reason string, opts ...DetectResultOption) DetectResult {
	return opt(true, "Opting in: "+reason, opts...)
}

// OptInAlways is used to always opt into the build process.
func OptInAlways(opts ...DetectResultOption) DetectResult {
	return OptIn("always enabled", opts...)
}

// OptInFileFound is used to opt into the build process based on file presence.
func OptInFileFound(file string, opts ...DetectResultOption) DetectResult {
	return OptIn("found "+file, opts...)
}

// OptInEnvSet is used to opt into the build process based on env var presence.
func OptInEnvSet(env string, opts ...DetectResultOption) DetectResult {
	return OptIn(fmt.Sprintf("%s set to %q", env, os.Getenv(env)), opts...)
}

// OptOut is used during the detect phase to opt out of the build process.
func OptOut(reason string, opts ...DetectResultOption) DetectResult {
	return opt(false, "Opting out: "+reason, opts...)
}

// OptOutFileNotFound is used to opt out of the build process based on file absence.
func OptOutFileNotFound(file string, opts ...DetectResultOption) DetectResult {
	return OptOut(file+" not found", opts...)
}

// OptOutEnvNotSet is used to opt out of the build process based on env var absence.
func OptOutEnvNotSet(env string, opts ...DetectResultOption) DetectResult {
	return OptOut(fmt.Sprintf("%s not set", env), opts...)
}

func opt(pass bool, reason string, opts ...DetectResultOption) DetectResult {
	r := &detectResult{
		reason: reason,
		result: libcnb.DetectResult{
			Pass: pass,
		},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}
