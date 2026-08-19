# The runner binary executes buildpacks for the Python language builder.
import argparse

from googlecloudplatform.buildpacks.commonbuildpacks import CommonBuildpacks
from googlecloudplatform.buildpacks.gcpbuildpack import GCPBuildpackRunner

# Buildpack libraries
import googlecloudplatform.buildpacks.python.appengine as python_appengine
import googlecloudplatform.buildpacks.python.functions_framework as python_functions_framework
import googlecloudplatform.buildpacks.python.functions_framework_compat as python_functions_framework_compat
import googlecloudplatform.buildpacks.python.link_runtime as python_link_runtime
import googlecloudplatform.buildpacks.python.missing_entrypoint as python_missing_entrypoint
import googlecloudplatform.buildpacks.python.pip as python_pip
import googlecloudplatform.buildpacks.python.poetry as python_poetry
import googlecloudplatform.buildpacks.python.runtime as python_runtime
import googlecloudplatform.buildpacks.python.uv as python_uv
import googlecloudplatform.buildpacks.python.webserver as python_webserver

def main():
    # Parse command line arguments
    parser = argparse.ArgumentParser(description='Run Python buildpacks')
    parser.add_argument('--buildpack', type=str, required=True,
                       help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', type=str, required=True,
                       choices=['detect', 'build'],
                       help='The phase to run: detect or build')
    args = parser.parse_args()

    # Register buildpack functions
    buildpacks = CommonBuildpacks()

    buildpacks["google.python.appengine"] = {
        "detect": python_appengine.detect,
        "build": python_appengine.build
    }
    buildpacks["google.python.functions-framework"] = {
        "detect": python_functions_framework.detect,
        "build": python_functions_framework.build
    }
    buildpacks["google.python.functions-framework-compat"] = {
        "detect": python_functions_framework_compat.detect,
        "build": python_functions_framework_compat.build
    }
    buildpacks["google.python.link-runtime"] = {
        "detect": python_link_runtime.detect,
        "build": python_link_runtime.build
    }
    buildpacks["google.python.missing-entrypoint"] = {
        "detect": python_missing_entrypoint.detect,
        "build": python_missing_entrypoint.build
    }
    buildpacks["google.python.pip"] = {
        "detect": python_pip.detect,
        "build": python_pip.build
    }
    buildpacks["google.python.poetry"] = {
        "detect": python_poetry.detect,
        "build": python_poetry.build
    }
    buildpacks["google.python.runtime"] = {
        "detect": python_runtime.detect,
        "build": python_runtime.build
    }
    buildpacks["google.python.webserver"] = {
        "detect": python_webserver.detect,
        "build": python_webserver.build
    }
    buildpacks["google.python.uv"] = {
        "detect": python_uv.detect,
        "build": python_uv.build
    }

    # Run the selected phase for the specified buildpack
    GCPBuildpackRunner(buildpacks, args.buildpack, args.phase).run()

if __name__ == "__main__":
    main()
