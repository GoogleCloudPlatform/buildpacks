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
Implements java/functions_framework buildpack.
The functions_framework buildpack copies the function framework into a layer, and adds it to a compiled function to make an executable app.
"""

import os
import re
import subprocess
from pathlib import Path

import google.cloud.platform.buildpacks.pkg.java as java
import google.cloud.platform.buildpacks.pkg.gcpbuildpack as gcp
from google.cloud.platform.buildpacks.internal.buildpacktest import BuildpackTestHelper

# Constants
LAYER_NAME = "functions-framework"
JAVA_FUNCTION_INVOKER_URL_BASE = (
    "https://maven-central.storage-download.googleapis.com/maven2/com/google/cloud/functions/invoker/java-function-invoker/"
)
DEFAULT_FRAMEWORK_VERSION = "1.4.1"
V2_FRAMEWORK_VERSION = "2.0.0"
FUNCTIONS_FRAMEWORK_URL_TEMPLATE = JAVA_FUNCTION_INVOKER_URL_BASE + "%s/java-function-invoker-%s.jar"
VERSION_KEY = "version"
INVOKER_MAIN = "com.google.cloud.functions.invoker.runner.Invoker"
IMPLEMENTATION_VERSION_KEY = "Implementation-Version"

# Regular expressions
FRAMEWORK_VERSION_REGEX = re.compile(r"java-function-invoker-((\d+\.)*\d+)")

def detect_fn(ctx: gcp.Context) -> gcp.DetectResult:
    """
    Detect function target environment variable.
    Returns OptIn if GOOGLE_FUNCTION_TARGET is set, otherwise OptOut.
    """
    if os.getenv("GOOGLE_FUNCTION_TARGET") is not None:
        return gcp.OptInEnvSet("GOOGLE_FUNCTION_TARGET")
    return gcp.OptOutEnvNotSet("GOOGLE_FUNCTION_TARGET")

def build_fn(ctx: gcp.Context) -> None:
    classpath_result, err = get_classpath(ctx)
    if err != None:
        raise err
    
    layer, err = ctx.layer(LAYER_NAME, gcp.LayerType.BUILD | gcp.LayerType.CACHE | gcp.LayerType.LAUNCH)
    if err != None:
        raise gcp.Error(f"Creating {LAYER_NAME} layer: {err}")
    
    ff_path, err = install_functions_framework(ctx, layer)
    if err != None:
        raise err
    
    layer.build_environment.override("FF_JAR_PATH", ff_path)
    
    if ctx.set_functions_env_vars(layer) != None:
        raise gcp.Error("Failed to set function environment variables")
    
    target = os.getenv("GOOGLE_FUNCTION_TARGET")
    if not target:
        raise gcp.UserError("Function target not specified")
        
    # Verify class exists
    try:
        subprocess.run(["javap", "-classpath", classpath_result, target], check=True)
    except subprocess.CalledProcessError as e:
        raise gcp.UserError(f"Build succeeded but did not produce the class {target}: {e}")
    
    # Create and set up launcher script
    create_launcher(ctx, layer.path)
    ctx.add_web_process([f"{layer.path}/launch.sh", "java", "-jar", ff_path, "--classpath", classpath_result])

def get_classpath(ctx: gcp.Context) -> str:
    if ctx.file_exists("pom.xml"):
        return maven_classpath(ctx)
    elif ctx.file_exists("build.gradle"):
        return gradle_classpath(ctx)
    else:
        jars = list(Path(".").glob("*.jar"))
        if len(jars) == 1:
            return str(jars[0])
        elif len(jars) > 1:
            raise gcp.UserError(f"Multiple jar files found: {', '.join(map(str, jars))}")
        else:
            files = list(Path(".").glob("*"))
            if not files:
                desc = "directory is empty"
            else:
                desc = f"directory contains: {', '.join(map(str, files))}"
            raise gcp.UserError(f"No pom.xml or jar file found; {desc}")

def maven_classpath(ctx: gcp.Context) -> str:
    mvn_cmd = java.maven_command(ctx)
    try:
        subprocess.run([mvn_cmd, "--batch-mode", "dependency:copy-dependencies", "-Dmdep.prependGroupId", "-DincludeScope=runtime"], check=True)
    except subprocess.CalledProcessError as e:
        raise gcp.Error(f"Failed to copy maven dependencies: {e}")
    
    result = subprocess.run([mvn_cmd, "help:evaluate", "-q", "-DforceStdout", "-Dexpression=project.build.finalName"], capture_output=True)
    artifact_name = result.stdout.strip().decode()
    if not artifact_name:
        raise gcp.UserError("Invalid project.build.finalName in pom.xml")
    
    jar_path = Path(f"target/{artifact_name}.jar")
    if not jar_path.exists():
        raise gcp.UserError(f"Expected jar file {jar_path} does not exist")
        
    return f"{jar_path}:target/dependency/*"

def gradle_classpath(ctx: gcp.Context) -> str:
    gradle_cmd = java.gradle_command(ctx)
    
    extra_tasks_path = Path(ctx.buildpack_root()) / "extra_tasks.gradle"
    with open(extra_tasks_path, 'r') as f:
        extra_tasks = f.read()
        
    try:
        build_gradle = Path("build.gradle")
        if not build_gradle.exists():
            raise gcp.Error(f"Could not find {build_gradle}")
            
        # Append extra tasks to build.gradle
        with open(build_gradle, 'a') as f:
            f.write(extra_tasks)
            
        subprocess.run([gradle_cmd, "--quiet", "_javaFunctionCopyAllDependencies"], check=True)
        
        result = subprocess.run([gradle_cmd, "--quiet", "_javaFunctionPrintJarTarget"], capture_output=True)
        jar_name = result.stdout.strip().decode()
        
        if not Path(jar_name).exists():
            raise gcp.UserError(f"Expected jar file {jar_name} does not exist")
            
        return f"{jar_name}:build/_javaFunctionDependencies/*"
    except subprocess.CalledProcessError as e:
        raise gcp.Error(f"Gradle command failed: {e}")

def create_launcher(ctx: gcp.Context, layer_path: str) -> None:
    launcher_source = Path(ctx.buildpack_root()) / "launch.sh"
    if not launcher_source.exists():
        raise gcp.Error(f"Launcher script not found at {launcher_source}")
        
    launcher_target = Path(layer_path) / "launch.sh"
    try:
        with open(launcher_source, 'r') as f:
            content = f.read()
            
        launcher_target.write_text(content)
        launcher_target.chmod(0o755)
    except Exception as e:
        raise gcp.Error(f"Failed to create launcher script: {e}")
