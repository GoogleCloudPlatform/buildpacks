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
//   buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.MyNewMetric).Increment(1)
const (
	ArNpmCredsGenCounterID CounterID = "1"
)
