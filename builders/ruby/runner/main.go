# The runner script executes buildpacks for the Ruby language builder.
import argparse
from gcp import main_runner

# Import each module's detect and build functions
from .nodejs.runtime.lib import detect_fn as nodejsruntime_detect, build_fn as nodejsruntime_build
from .nodejs.yarn.lib import detect_fn as nodejyarn_detect, build_fn as nodejyarn_build
from .ruby.appengine.lib import detect_fn as rubyappengine_detect, build_fn as rubyappengine_build
from .ruby.appengine_validation.lib import detect_fn as rubyappenginevalidation_detect, build_fn as rubyappenginevalidation_build
from .ruby.bundle.lib import detect_fn as rubybundle_detect, build_fn as rubybundle_build
from .ruby.flex_entrypoint.lib import detect_fn as rubyflexentrypoint_detect, build_fn as rubyflexentrypoint_build
from .ruby.functions_framework.lib import detect_fn as rubyfunctionsframework_detect, build_fn as rubyfunctionsframework_build
from .ruby.missing_entrypoint.lib import detect_fn as rubymissingentrypoint_detect, build_fn as rubymissingentrypoint_build
from .ruby.rails.lib import detect_fn as rubyrails_detect, build_fn as rubyrails_build
from .ruby.runtime.lib import detect_fn as rubyruntime_detect, build_fn as rubyruntime_build
from .ruby.rubygems.lib import detect_fn as rubyrubygems_detect, build_fn as rubyrubygems_build

# Register buildpack functions here
buildpacks = {
    "google.nodejs.runtime": {
        "detect": nodejsruntime_detect,
        "build": nodejsruntime_build
    },
    "google.nodejs.yarn": {
        "detect": nodejyarn_detect,
        "build": nodejyarn_build
    },
    "google.ruby.appengine": {
        "detect": rubyappengine_detect,
        "build": rubyappengine_build
    },
    "google.ruby.appengine-validation": {
        "detect": rubyappenginevalidation_detect,
        "build": rubyappenginevalidation_build
    },
    "google.ruby.bundle": {
        "detect": rubybundle_detect,
        "build": rubybundle_build
    },
    "google.ruby.flex-entrypoint": {
        "detect": rubyflexentrypoint_detect,
        "build": rubyflexentrypoint_build
    },
    "google.ruby.functions-framework": {
        "detect": rubyfunctionsframework_detect,
        "build": rubyfunctionsframework_build
    },
    "google.ruby.missing-entrypoint": {
        "detect": rubymissingentrypoint_detect,
        "build": rubymissingentrypoint_build
    },
    "google.ruby.rails": {
        "detect": rubyrails_detect,
        "build": rubyrails_build
    },
    "google.ruby.runtime": {
        "detect": rubyruntime_detect,
        "build": rubyruntime_build
    },
    "google.ruby.rubygems": {
        "detect": rubyrubygems_detect,
        "build": rubyrubygems_build
    }
}

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Run Ruby buildpacks.')
    parser.add_argument('--buildpack', required=True, help='The ID of the buildpack to run')
    parser.add_argument('--phase', required=True, choices=['detect', 'build'], help='The phase to run: detect or build')
    args = parser.parse_args()
    main_runner(buildpacks, args.buildpack, args.phase)
