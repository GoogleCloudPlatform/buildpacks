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

// CounterID is an index to Counter
type CounterID string

// **********************************************
// ** ID VALUES MUST NEVER CHANGE OR BE REUSED **
// **********************************************
// Changing these values will cause metric values to be interpreted as the wrong
// type when the producer and consumer use different orderings.  Metric IDs may
// be deleted, but the metric's id must be reserved.

// The CounterID consts below define new counter metrics that can be recorded by
// instrumenting code.  To add a new metric, add a new const CounterID below and
// the buildermetrics package will be able to track a new Counter metric.
//
// Intended usage:
//
//	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.MyNewMetric).Increment(1)
const (
	ArNpmCredsGenCounterID                CounterID = "1"
	NpmGcpBuildUsageCounterID             CounterID = "2"
	NpmBuildUsageCounterID                CounterID = "3"
	NpmGoogleNodeRunScriptsUsageCounterID CounterID = "4"
)

var (
	counterDescriptors = map[CounterID]Descriptor{
		ArNpmCredsGenCounterID: Descriptor{
			"npm_artifact_registry_creds_generated",
			"The number of artifact registry credentials generated for NPM",
		},
		NpmGcpBuildUsageCounterID: Descriptor{
			"npm_gcp_build_script_uses",
			"The number of times the gcp-build script is used by npm developers",
		},
		NpmBuildUsageCounterID: Descriptor{
			"npm_build_script_uses",
			"The number of times an npm build script is used by npm developers",
		},
		NpmGoogleNodeRunScriptsUsageCounterID: Descriptor{
			"npm_google_node_run_script_uses",
			"The number of times the GOOGLE_NODE_RUN_SCRIPTS env var is used by npm developers",
		},
	}
)

// Descriptor provides details about a metric
type Descriptor struct {
	Name        string
	Description string
}

// Descriptor returns the Descriptor for a CounterID
func (c CounterID) Descriptor() (Descriptor, error) {
	desc, ok := counterDescriptors[c]
	if !ok {
		return Descriptor{}, fmt.Errorf("Descriptor for CounterID %q not found", c)
	}
	if desc.Name == "" || desc.Description == "" {
		return Descriptor{}, fmt.Errorf("Descriptor %q (for CounterID %q) must have both a Name and a Description", desc, c)
	}
	return desc, nil

}
