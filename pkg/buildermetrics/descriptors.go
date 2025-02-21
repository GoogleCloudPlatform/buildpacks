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

// The metricID consts below define new metrics that can be recorded by
// instrumenting code.  To add a new metric, add a new const metricID below and
// the buildermetrics package will be able to track a new metric.
//
// Intended usage:
//
//	buildermetrics.GlobalBuilderMetrics().GetCounter(buildermetrics.MyNewMetric).Add(1)
//	buildermetrics.GlobalBuilderMetrics().GetFloatDP(buildermetrics.MyNewMetric2).Add(1.0)
const (
	ArNpmCredsGenCounterID                MetricID = "1"
	NpmGcpBuildUsageCounterID             MetricID = "2"
	NpmBuildUsageCounterID                MetricID = "3"
	NpmGoogleNodeRunScriptsUsageCounterID MetricID = "4"
	PipVendorDependenciesCounterID        MetricID = "5"
	NpmNodeModulesCounterID               MetricID = "6"
	NpmVendorDependenciesCounterID        MetricID = "7"
	NpmInstallLatencyID                   MetricID = "8"
	ComposerInstallLatencyID              MetricID = "9"
	PipInstallLatencyID                   MetricID = "10"
	JavaGAEWebXMLConfigUsageCounterID     MetricID = "11"
	JavaGAESessionsEnabledCounterID       MetricID = "12"
)

var (
	descriptors = map[MetricID]Descriptor{
		ArNpmCredsGenCounterID: newDescriptor(
			ArNpmCredsGenCounterID,
			"npm_artifact_registry_creds_generated",
			"The number of artifact registry credentials generated for NPM",
		),
		NpmGcpBuildUsageCounterID: newDescriptor(
			NpmGcpBuildUsageCounterID,
			"npm_gcp_build_script_uses",
			"The number of times the gcp-build script is used by npm developers",
		),
		NpmBuildUsageCounterID: newDescriptor(
			NpmBuildUsageCounterID,
			"npm_build_script_uses",
			"The number of times an npm build script is used by npm developers",
		),
		NpmGoogleNodeRunScriptsUsageCounterID: newDescriptor(
			NpmGoogleNodeRunScriptsUsageCounterID,
			"npm_google_node_run_script_uses",
			"The number of times the GOOGLE_NODE_RUN_SCRIPTS env var is used by npm developers",
		),
		PipVendorDependenciesCounterID: newDescriptor(
			PipVendorDependenciesCounterID,
			"vendor_pip_dependencies_uses",
			"The number of times GOOGLE_VENDOR_PIP_DEPENDENCIES is used by developers",
		),
		NpmNodeModulesCounterID: newDescriptor(
			NpmNodeModulesCounterID,
			"npm_node_modules_uses",
			"The number of times node_modules directory exist in source code",
		),
		NpmVendorDependenciesCounterID: newDescriptor(
			NpmVendorDependenciesCounterID,
			"vendor_npm_dependencies_uses",
			"The number of times GOOGLE_VENDOR_NPM_DEPENDENCIES is used by developers",
		),
		NpmInstallLatencyID: newDescriptor(
			NpmInstallLatencyID,
			"npm_install_latency",
			"The latency for executions of `npm install`",
		),
		ComposerInstallLatencyID: newDescriptor(
			ComposerInstallLatencyID,
			"composer_install_latency",
			"The latency for executions of `composer install`",
		),
		PipInstallLatencyID: newDescriptor(
			PipInstallLatencyID,
			"pip_install_latency",
			"The latency for executions of `pip install`",
		),
		JavaGAEWebXMLConfigUsageCounterID: newDescriptor(
			JavaGAEWebXMLConfigUsageCounterID,
			"java_gae_web_xml_config_uses",
			"The number of times the appengine-web.xml is used by developers",
		),
		JavaGAESessionsEnabledCounterID: newDescriptor(
			JavaGAESessionsEnabledCounterID,
			"java_gae_session_handler_uses",
			"The number of times the session handler is used by developers",
		),
	}
)
