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

"""Implements Java GraalVM Native Image buildpack."""

import os
import re
import subprocess
from typing import List, Optional, Tuple

import gcpbuildpack as gcp
import java_utils
import libcnb

INVOKER_MAIN = "com.google.cloud.functions.invoker.runner.Invoker"

REQUIRES_GRAALVM = [libcnb.BuildPlanRequire(name="graalvm")]
PLAN_REQUIRES = libcnb.BuildPlan(requires=REQUIRES_GRAALVM)

def detect(context: gcp.Context) -> Tuple[gcp.DetectResult, Exception]:
    """Detect function for the buildpack."""
    return gcp.opt_in_always(gcp.with_build_plans(PLAN_REQUIRES)), None

def build(context: gcp.Context) -> Exception:
    """Build function for the buildpack."""
    entrypoint, err = create_image(context)
    if err is not None:
        return err
    context.add_web_process(entrypoint)
    return None

def create_image(context: gcp.Context) -> Tuple[List[str], Exception]:
    """Creates a native image and returns its entrypoint."""
    pom, err = parse_pom_file(context)
    if err is not None:
        return None, Exception(f"parsing pom file: {err}")
    if pom is None:
        return build_default(context)
    
    function_target = os.getenv("FUNCTION_TARGET")
    if function_target is not None:
        return build_functions_framework(context, function_target, pom)

    build_profile, found = find_native_build_profile(context, pom)
    if found:
        return build_maven(context, build_profile)
    
    if spring_boot_plugin_defined(context, pom):
        entrypoint, err = build_springboot(context)
        if err is not None:
            return None, err
        elif entrypoint is not None:
            return entrypoint, None
    
    return build_default(context)

def build_default(context: gcp.Context) -> Tuple[List[str], Exception]:
    """Builds a native image for standard Java apps."""
    jar_path, err = java_utils.executable_jar(context)
    if err is not None:
        return None, Exception(f"finding executable jar: {err}")
    return build_command_line(context, ["-jar", jar_path])

def build_command_line(
    context: gcp.Context,
    build_args: List[str]
) -> Tuple[List[str], Exception]:
    """Builds a native image via command line."""
    ni_dir = os.path.join(context.temp_dir(), "native-image")
    temp_image_path = os.path.join(ni_dir, "native-app")

    user_args = os.getenv("NATIVE_IMAGE_BUILD_ARGS", "")
    command = f"native-image --no-fallback --no-server -H:+StaticExecutableWithDynamicLibC {user_args} {' '.join(build_args)} {temp_image_path}"
    
    try:
        subprocess.run(["bash", "-c", command], check=True)
    except subprocess.CalledProcessError as e:
        return None, Exception(str(e))
    
    native_layer = context.layer("native-image", gcp.LayerType.LAUNCH)
    final_image = os.path.join(native_layer.path, "bin", "native-app")
    
    try:
        os.makedirs(final_image, exist_ok=True)
        os.rename(temp_image_path, final_image)
    except Exception as e:
        return None, Exception(str(e))
    
    return [final_image], None

def build_maven(
    context: gcp.Context,
    build_profile: str
) -> Tuple[List[str], Exception]:
    """Builds a native image using Maven."""
    mvn_cmd = java_utils.mvn(context)
    command = [mvn_cmd, "package", "-DskipTests", "--batch-mode", "-Dhttp.keepAlive=false"]
    
    if build_profile:
        command.append(f"-P{build_profile}")
    
    try:
        subprocess.run(command, check=True)
    except subprocess.CalledProcessError as e:
        return None, Exception(str(e))
    
    image_path, err = find_native_executable(context)
    if err is not None:
        return None, err
    return [image_path], None

def parse_pom_file(context: gcp.Context) -> Tuple[java_utils.MavenProject, Exception]:
    """Parses the pom.xml file."""
    pom_exists, err = context.file_exists("pom.xml")
    if err is not None:
        return None, err
    if not pom_exists:
        return None, None
    
    tmp_dir = os.path.join(context.temp_dir(), "native-image-maven")
    effective_pom_path = os.path.join(tmp_dir, "project_effective_pom.xml")
    
    mvn_cmd = java_utils.mvn(context)
    try:
        subprocess.run([
            mvn_cmd,
            "help:effective-pom",
            "--batch-mode",
            f"-Dhttp.keepAlive=false",
            f"-Doutput={effective_pom_path}"
        ], check=True)
    except subprocess.CalledProcessError as e:
        return None, Exception(str(e))
    
    try:
        with open(effective_pom_path, "r") as f:
            pom_content = f.read()
        project = java_utils.parse_pom(pom_content)
        return project, None
    except Exception as e:
        context.warn(f"A pom.xml was found but unable to be parsed: {e}")
        return None, None

def find_native_build_profile(
    context: gcp.Context,
    project: java_utils.MavenProject
) -> Tuple[str, bool]:
    """Finds the native build profile in the Maven project."""
    for profile in project.profiles:
        for plugin in profile.plugins:
            if plugin.group_id == "org.graalvm.nativeimage" and \
               plugin.artifact_id == "native-image-maven-plugin":
                return profile.id, True
    context.log("Did not find a native-image-plugin defined in the pom.xml")
    return "", False

def spring_boot_plugin_defined(
    context: gcp.Context,
    project: java_utils.MavenProject
) -> bool:
    """Checks if the Spring Boot Maven plugin is defined."""
    for plugin in project.plugins:
        if plugin.group_id == "org.springframework.boot" and \
           plugin.artifact_id == "spring-boot-maven-plugin":
            return True
    context.log("Did not find a spring-boot-maven-plugin defined in the pom.xml")
    return False

def build_springboot(context: gcp.Context) -> Tuple[List[str], Exception]:
    """Builds a native image for Spring Boot applications."""
    classpath, main_class, err = classpath_and_main_from_springboot(context)
    if err is not None:
        return None, err
    elif classpath == "" or main_class == "":
        return None, None
    return build_command_line(context, ["--class-path", classpath, main_class])

def find_native_executable(context: gcp.Context) -> Tuple[str, Exception]:
    """Finds the native executable in the target directory."""
    try:
        files = context.read_dir("target")
    except Exception as e:
        return "", Exception(str(e))
    
    executables = []
    for file_info in files:
        if not os.path.isdir(file_info.name) and \
           (os.stat(file_info.path).st_mode & 0o111):
            executables.append(os.path.join("target", file_info.name))
    
    if len(executables) != 1:
        return "", Exception(f"Expected exactly 1 executable in target/, found: {executables}")
    return executables[0], None

def classpath_and_main_from_springboot(
    context: gcp.Context
) -> Tuple[str, str, Exception]:
    """Extracts classpath and main class from a Spring Boot JAR."""
    jar_path, err = java_utils.executable_jar(context)
    if err is not None:
        context.warn(f"Spring Boot project assumed but no main executable JAR found: {err}")
        return "", "", None
    
    start_class, err = java_utils.find_manifest_value_from_jar(jar_path, "Start-Class")
    if err is not None:
        return "", "", Exception(f"Fetching manifest value from JAR: {jar_path}")
    elif start_class == "":
        context.warn(f"Spring Boot project assumed but Start-Class undefined in executable JAR: {jar_path}")
        return "", "", None
    
    exploded_jar_dir = os.path.join(context.temp_dir(), "exploded-jar")
    try:
        subprocess.run(["unzip", "-q", jar_path, "-d", exploded_jar_dir], check=True)
    except subprocess.CalledProcessError as e:
        return "", "", Exception(str(e))
    
    classes_dir = os.path.join(exploded_jar_dir, "BOOT-INF", "classes")
    libs_dir = os.path.join(exploded_jar_dir, "BOOT-INF", "lib", "*")
    classpath = ":".join([exploded_jar_dir, classes_dir, libs_dir])
    
    return classpath, start_class, None

def build_functions_framework(
    context: gcp.Context,
    function_target: str,
    project: java_utils.MavenProject
) -> Tuple[List[str], Exception]:
    """Builds a native image for the Functions Framework."""
    classpath, err = create_functions_classpath(context, project)
    if err is not None:
        return None, err
    
    entrypoint, err = build_command_line(context, ["-cp", classpath, INVOKER_MAIN])
    if err is not None:
        return None, err
    functions_framework_entrypoint = entrypoint + ["--target", function_target]
    return functions_framework_entrypoint, None

def create_functions_classpath(
    context: gcp.Context,
    project: java_utils.MavenProject
) -> Tuple[str, Exception]:
    """Creates the classpath for the Functions Framework."""
    jar_name = f"{project.artifact_id}-{project.version}.jar"
    application_jar = os.path.join("target", jar_name)
    
    exists, err = context.file_exists(application_jar)
    if err is not None:
        return "", Exception(str(err))
    elif not exists:
        return "", Exception(f"Finding application JAR: {application_jar}")
    
    dependencies_dir = os.path.join("target", "dependency", "*")
    classpath = ":".join([
        os.getenv(" FUNCTIONS_FRAMEWORK_JAR_PATH"),
        application_jar,
        dependencies_dir
    ])
    return classpath, None
