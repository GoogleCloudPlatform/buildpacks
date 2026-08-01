# Complete refactored code here
"""
Implements java/appengine buildpack.
The appengine buildpack sets the image entrypoint.

Copyright 2025 Google LLC Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
"""

import os
import logging
from typing import Any, Dict, List, Optional

import google.cloud.buildpacks as gcp
import google.cloud.buildpacks.appengine as appengine
import google.cloud.buildpacks.appstart as appstart
import google.cloud.buildpacks.buildermetrics as buildermetrics
import google.cloud.buildpacks.env as env
import google.cloud.buildpacks.java as java
import google.cloud.buildpacks.runtime as runtime

class EEConfig:
    def __init__(self, ee8_deletions: List[str], ee10_deletions: List[str], 
                 ee11_deletions: List[str], default_ee_version: str):
        self.ee8_deletions = ee8_deletions
        self.ee10_deletions = ee10_deletions
        self.ee11_deletions = ee11_deletions
        self.default_ee_version = default_ee_version

JETTY_FILES_TO_DELETE: Dict[str, EEConfig] = {
    "java25": EEConfig(
        ee8_deletions=["runtime-shared-jetty121-ee11.jar"],
        ee10_deletions=[],
        ee11_deletions=["runtime-shared-jetty121-ee8.jar"],
        default_ee_version="EE11"
    )
}

def detect_fn(context: gcp.Context) -> Optional[gcp.DetectResult]:
    if env.is_gae():
        return appengine.opt_in_target_platform_gae()
    return appengine.opt_out_target_platform_not_gae()

def build_fn(context: gcp.Context) -> None:
    jetty_entrypoint = _get_jetty_entrypoint(context)
    entrypoint_generator = (
        lambda ctx: (jetty_entrypoint, None) if jetty_entrypoint 
        else _get_custom_entrypoint(ctx)
    )
    appengine.build(context, "java", entrypoint_generator)

def _get_jetty_entrypoint(context: gcp.Context) -> Optional[appstart.Entrypoint]:
    web_xml_exists = context.file_exists("WEB-INF", "appengine-web.xml")
    if not web_xml_exists:
        return None

    metrics = buildermetrics.GlobalBuilderMetrics()
    metrics.get_counter(buildermetrics.JavaGAEWebXMLConfigUsageCounterID).increment(1)

    jetty_layer, err = _process_app_engine_web_xml(context)
    if err:
        raise ValueError(f"Error processing appengine-web.xml: {err}")

    if jetty_layer:
        context.log.info("WAR packaging detected and injecting the embedded web-server dependencies at build time")
        return appstart.Entrypoint(
            type=appstart.EntrypointGenerated,
            command=f"serve WEB-INF/appengine-web.xml {jetty_layer}"
        )
    
    return appstart.Entrypoint(
        type=appstart.EntrypointGenerated,
        command="serve WEB-INF/appengine-web.xml"
    )

def _get_custom_entrypoint(context: gcp.Context) -> appstart.Entrypoint:
    executable, err = java.executable_jar(context)
    if err:
        raise ValueError(f"Finding executable jar: {err}")

    return appstart.Entrypoint(
        type=appstart.EntrypointGenerated,
        command=f"serve {executable}"
    )

def _process_app_engine_web_xml(context: gcp.Context) -> (str, bool):
    full_path = os.path.join(context.application_root(), "WEB-INF/appengine-web.xml")
    appengine_web_xml, err = context.read_file(full_path)
    if err:
        raise ValueError(f"Error reading appengine-web.xml: {err}")

    parsed_app, err = java.parse_app_engine_web_xml(appengine_web_xml)
    if err:
        raise ValueError(f"Error parsing appengine-web.xml: {err}")

    if parsed_app.sessions_enabled:
        metrics = buildermetrics.GlobalBuilderMetrics()
        metrics.get_counter(buildermetrics.JavaGAESessionsEnabledCounterID).increment(1)

    if parsed_app.runtime in supported_jetty_build_time_versions():
        return _add_jetty_at_build_time(context, parsed_app), True
    
    return "", False

def supported_jetty_build_time_versions() -> List[str]:
    return ["java25"]

def _add_jetty_at_build_time(context: gcp.Context, 
                            appengine_web_xml_app: java.AppEngineWebXMLApp) -> str:
    jetty_layer, err = context.layer("java_runtime", gcp.LayerType.LAUNCH)
    if err:
        raise ValueError(f"Creating layer: {err}")

    repo_path = appengine_web_xml_app.runtime
    _, err = runtime.install_tarball_if_not_cached(
        context, runtime.Jetty, "", jetty_layer.path
    )
    if err:
        raise ValueError(f"Error installing jetty artifacts: {err}")

    context.log.info(f"Successfully installed Jetty for {repo_path} at build time from AR.")

    _handle_runtime_jetty_files(context, appengine_web_xml_app, jetty_layer.path)
    return jetty_layer.path

def _handle_runtime_jetty_files(context: gcp.Context,
                                appengine_web_xml_app: java.AppEngineWebXMLApp,
                                jetty_root: str) -> None:
    config = JETTY_FILES_TO_DELETE.get(appengine_web_xml_app.runtime)
    if not config:
        return

    ee_version, err = _extract_ee_version(
        appengine_web_xml_app, config.default_ee_version, context
    )
    if err:
        raise ValueError(err)

    files_to_delete = []
    if ee_version == "EE8":
        files_to_delete = config.ee8_deletions
    elif ee_version == "EE11":
        files_to_delete = config.ee11_deletions

    if not files_to_delete:
        return

    for root, _, files in os.walk(jetty_root):
        for file_name in files:
            full_path = os.path.join(root, file_name)
            if (file_name in files_to_delete or 
                not file_name.endswith(".jar")):
                try:
                    os.remove(full_path)
                except Exception as e:
                    context.log.warning(
                        f"Failed to delete file {full_path}: {e}"
                    )

def _extract_ee_version(appengine_web_xml_app: java.AppEngineWebXMLApp,
                       default_ee_version: str, 
                       context: gcp.Context) -> (str, Optional[str]):
    true_count = 0
    ee_version = ""
    error = None

    for prop in appengine_web_xml_app.system_properties:
        if prop.name == "appengine.use.EE11" and prop.value.lower() == "true":
            true_count += 1
            ee_version = "EE11"
        elif prop.name == "appengine.use.EE8" and prop.value.lower() == "true":
            true_count += 1
            ee_version = "EE8"
        elif prop.name == "appengine.use.EE10" and prop.value.lower() == "true":
            true_count += 1
            ee_version = "EE10"
            error = "appengine.use.EE10 is not supported in Jetty121"

        if true_count > 1:
            return "", "Only one of appengine.use.EE8, appengine.use.EE10, or appengine.use.EE11 can be true"

    if true_count == 0:
        ee_version = default_ee_version
        context.log.info(
            f"No appengine.use.* property found in appengine-web.xml, "
            f"using default EE version: {ee_version}"
        )

    return ee_version, error
