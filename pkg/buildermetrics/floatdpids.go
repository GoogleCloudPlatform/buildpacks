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

package buildermetrics

import "fmt"

// FloatDPID is an index to FloatDP
type FloatDPID string

// **********************************************
// ** ID VALUES MUST NEVER CHANGE OR BE REUSED **
// **********************************************
// Changing these values will cause metric values to be interpreted as the wrong
// type when the producer and consumer use different orderings.  Metric IDs may
// be deleted, but the metric's id must be reserved.

// The FloatDPID consts below define new float64 distribution point that can be recorded by
// instrumenting code.  To add a new metric, add a new const FloatDPID below and
// the buildermetrics package will be able to track a new FloatDP metric.
//
// Intended usage:
//
//	buildermetrics.GlobalBuilderMetrics().GetFloatDP(buildermetrics.MyNewMetric).Add(3.76)
const (
	NpmInstallLatencyID      FloatDPID = "1"
	ComposerInstallLatencyID FloatDPID = "2"
	PipInstallLatencyID      FloatDPID = "3"
)

var (
	floatDPDescriptors = map[FloatDPID]Descriptor{
		NpmInstallLatencyID: Descriptor{
			"npm_install_latency",
			"The latency for executions of `npm install`",
		},
		ComposerInstallLatencyID: Descriptor{
			"composer_install_latency",
			"The latency for executions of `composer install`",
		},
		PipInstallLatencyID: Descriptor{
			"pip_install_latency",
			"The latency for executions of `pip install`",
		},
	}
)

// Descriptor returns the Descriptor for a FloatDPID
func (f FloatDPID) Descriptor() (Descriptor, error) {
	desc, ok := floatDPDescriptors[f]
	if !ok {
		return Descriptor{}, fmt.Errorf("Descriptor for FloatDPID %q not found", f)
	}
	if desc.Name == "" || desc.Description == "" {
		return Descriptor{}, fmt.Errorf("Descriptor %q (for FloatDPID %q) must have both a Name and a Description", desc, f)
	}
	return desc, nil
}
