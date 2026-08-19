# The runner binary executes buildpacks for the Java language builder.
import argparse

from gcp_buildpacks import gcp_main_runner  # Assuming this is the equivalent of gcp.MainRunner in Go

# Buildpack libraries
from google.java.appengine import detect as java_appengine_detect, build as java_appengine_build
from google.java.clear_source import detect as java_clearsource_detect, build as java_clearsource_build
from google.java.entrypoint import detect as java_entrypoint_detect, build as java_entrypoint_build
from google.java.exploded_jar import detect as java_explodedjar_detect, build as java_explodedjar_build
from google.java.functions_framework import detect as java_functionsframework_detect, build as java_functionsframework_build
from google.java.gradle import detect as java_gradle_detect, build as java_gradle_build
from google.java.maven import detect as java_maven_detect, build as java_maven_build
from google.java.runtime import detect as java_runtime_detect, build as java_runtime_build
from google.java.spring_boot import detect as java_springboot_detect, build as java_springboot_build

# Register buildpack functions here
buildpacks = {
    "google.java.appengine": {
        "detect": java_appengine_detect,
        "build": java_appengine_build
    },
    "google.java.clear-source": {
        "detect": java_clearsource_detect,
        "build": java_clearsource_build
    },
    "google.java.entrypoint": {
        "detect": java_entrypoint_detect,
        "build": java_entrypoint_build
    },
    "google.java.exploded-jar": {
        "detect": java_explodedjar_detect,
        "build": java_explodedjar_build
    },
    "google.java.functions-framework": {
        "detect": java_functionsframework_detect,
        "build": java_functionsframework_build
    },
    "google.java.gradle": {
        "detect": java_gradle_detect,
        "build": java_gradle_build
    },
    "google.java.maven": {
        "detect": java_maven_detect,
        "build": java_maven_build
    },
    "google.java.runtime": {
        "detect": java_runtime_detect,
        "build": java_runtime_build
    },
    "google.java.spring-boot": {
        "detect": java_springboot_detect,
        "build": java_springboot_build
    }
}

def main():
    parser = argparse.ArgumentParser(description='Run Java buildpacks')
    parser.add_argument('--buildpack', type=str, required=True,
                       help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', type=str, required=True,
                       help='The phase to run: "detect" or "build"')

    args = parser.parse_args()

    if not args.buildpack:
        print("Error: --buildpack is required")
        return
    if not args.phase:
        print("Error: --phase is required")
        return

    # Assuming gcp_main_runner expects the buildpacks dict, buildpack ID, and phase
    gcp_main_runner(buildpacks, args.buildpack, args.phase)

if __name__ == "__main__":
    main()
