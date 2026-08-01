# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Package lib implements the java/spring-boot buildpack.
"""

import os
import logging
from typing import Optional

import packaging.version as semver
from cmd.java import buildermetrics
from cmd.java import gcpbuildpack as gcp
from cmd.java import java


SPRING_BOOT_VERSION_MANIFEST = "Spring-Boot-Version"


def detect_fn(ctx: gcp.Context) -> tuple[gcp.DetectResult, Exception]:
    """
    Detects if the application is a Spring Boot application.
    """
    runtime_version = os.environ.get("GOOGLE_RUNTIME_VERSION")
    
    if runtime_version:
        cleaned_version_str = runtime_version.replace("_", "-", 1)
        try:
            v = semver.parse(cleaned_version_str)
            if v.major < 17:
                return gcp.OptOut(f"Runtime version {runtime_version} is less than java17."), None
        except ValueError as err:
            return gcp.OptOut(f"Failed to parse runtime version '{runtime_version}' as semver: {err}"), None
    
    logging.info("Checking for packaged JAR")
    spring_boot_version, _ = spring_boot_version_in_manifest(ctx)
    
    if spring_boot_version:
        logging.info(f"Detected Spring Boot version {spring_boot_version} in manifest")
        return gcp.OptIn("Opted in, Spring Boot version found in manifest."), None
    
    logging.info("Checking for Spring Boot in pom.xml. JAR file not present at detect time.")
    
    if os.path.exists("pom.xml"):
        project, err = parse_pom_file(ctx)
        
        if err is None and project:
            if spring_boot_starter_defined(ctx, project) or spring_boot_plugin_defined(ctx, project):
                logging.info("Detected Spring Boot in pom.xml")
                return gcp.OptIn("Opted in, Spring Boot detected."), None
    
    return gcp.OptOut("Not a Spring Boot project"), None


def build_fn(ctx: gcp.Context) -> Exception:
    """
    Adds to the metric if it is a Spring Boot application.
    """
    spring_boot_version, _ = spring_boot_version_in_manifest(ctx)
    
    if spring_boot_version:
        buildermetrics.global_builder_metrics().get_counter(buildermetrics.JavaSpringBootUsageCounterID).increment(1)
        logging.info(f"Detected Spring Boot version {spring_boot_version}. Incremented counter.")
    else:
        logging.info("Not a Spring Boot application (Spring-Boot-Version not found in manifest). Skipping Spring Boot buildpack logic.")
    
    return None


def spring_boot_version_in_manifest(ctx: gcp.Context) -> tuple[str, Exception]:
    """
    Returns the Spring Boot version in the manifest of the application JAR.
    """
    app_jar, err = java.executable_jar(ctx)
    
    if err:
        logging.error(f"Error finding executable jar: {err}")
        return "", None
    
    if not app_jar:
        logging.info("No executable JAR found. Skipping Spring Boot buildpack logic.")
        return "", None
    
    logging.info(f"Found potential application JAR: {app_jar}")
    
    spring_boot_version, err = java.find_manifest_value_from_jar(app_jar, SPRING_BOOT_VERSION_MANIFEST)
    
    if err:
        logging.error(f"Error reading manifest from jar {app_jar}: {err}")
        return "", None
    
    return spring_boot_version, None


def parse_pom_file(ctx: gcp.Context) -> tuple[Optional[java.MavenProject], Exception]:
    """
    Returns a parsed pom.xml if it exists.
    """
    if not os.path.exists("pom.xml"):
        return None, None
    
    try:
        with open("pom.xml", "r") as f:
            pom_content = f.read()
        
        project = java.parse_pom_file(pom_content)
        return project, None
    except Exception as err:
        logging.error(f"Failed to parse effective pom: {err}")
        return None, err


def spring_boot_plugin_defined(ctx: gcp.Context, project: java.MavenProject) -> bool:
    """
    Checks if the spring-boot-maven-plugin is defined.
    """
    for plugin in project.plugins:
        if plugin.group_id == "org.springframework.boot" and plugin.artifact_id == "spring-boot-maven-plugin":
            return True
    
    logging.info("Did not find a spring-boot-maven-plugin defined in the pom.xml")
    return False


def spring_boot_starter_defined(ctx: gcp.Context, project: java.MavenProject) -> bool:
    """
    Checks if the spring-boot-starter is defined.
    """
    for dependency in project.dependencies:
        if "spring-boot-starter" in dependency.artifact_id:
            return True
    
    logging.info("Did not find a spring-boot-starter defined in the pom.xml")
    return False
