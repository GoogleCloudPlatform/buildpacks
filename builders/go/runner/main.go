# The runner script executes buildpacks for the Go language builder.
import argparse

from googlecloudplatform.buildpacks.common import common_buildpacks
from googlecloudplatform.buildpacks.gcp import gcp_buildpack

# Buildpack modules
import googlecloudplatform.buildpacks.go.appengine as goappengine
import googlecloudplatform.buildpacks.go.appengine_gomod as goappenginegomod
import googlecloudplatform.buildpacks.go.appengine_gopath as goappenginegopath
import googlecloudplatform.buildpacks.go.build as gobuild
import googlecloudplatform.buildpacks.go.clear_source as goclearsource
import googlecloudplatform.buildpacks.go.flex_gomod as goflexgomod
import googlecloudplatform.buildpacks.go.functions_framework as gofunctionsframework
import googlecloudplatform.buildpacks.go.gomod as gogomod
import googlecloudplatform.buildpacks.go.gopath as gogopath
import googlecloudplatform.buildpacks.go.legacy_worker as golegacyworker
import googlecloudplatform.buildpacks.go.runtime as goruntime

def main():
    parser = argparse.ArgumentParser(description='Run Go buildpacks.')
    parser.add_argument('--buildpack', required=True, help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', required=True, choices=['detect', 'build'], help='The phase to run: detect or build')
    args = parser.parse_args()

    # Register buildpack functions
    buildpacks = {
        "google.go.appengine": {
            "detect": goappengine.detect,
            "build": goappengine.build
        },
        "google.go.appengine-gomod": {
            "detect": goappenginegomod.detect,
            "build": goappenginegomod.build
        },
        "google.go.flex-gomod": {
            "detect": goflexgomod.detect,
            "build": goflexgomod.build
        },
        "google.go.appengine-gopath": {
            "detect": goappenginegopath.detect,
            "build": goappenginegopath.build
        },
        "google.go.build": {
            "detect": gobuild.detect,
            "build": gobuild.build
        },
        "google.go.clear-source": {
            "detect": goclearsource.detect,
            "build": goclearsource.build
        },
        "google.go.functions-framework": {
            "detect": gofunctionsframework.detect,
            "build": gofunctionsframework.build
        },
        "google.go.gomod": {
            "detect": gogomod.detect,
            "build": gogomod.build
        },
        "google.go.gopath": {
            "detect": gogopath.detect,
            "build": gogopath.build
        },
        "google.go.legacy-worker": {
            "detect": golegacyworker.detect,
            "build": golegacyworker.build
        },
        "google.go.runtime": {
            "detect": goruntime.detect,
            "build": goruntime.build
        }
    }

    # Get buildpack details
    selected_buildpack = args.buildpack
    phase = args.phase

    if selected_buildpack not in buildpacks:
        raise ValueError(f"Buildpack {selected_buildpack} is not registered.")

    buildpack = buildpacks[selected_buildpack]

    if phase == 'detect':
        buildpack['detect']()
    elif phase == 'build':
        buildpack['build']()

if __name__ == "__main__":
    main()
