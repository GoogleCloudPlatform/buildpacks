import argparse

from google.cloud import buildpacks_v2
from google.cloud.buildpacks_v2.types import buildpack as buildpack_types

# Buildpack libraries
import phpappengine
import phpcloudfunctions
import phpcomposer
import phpcomposergcpbuild
import phpcomposerinstall
import phpfunctionsframework
import phpruntime
import phpsupervisor
import phpwebconfig
import pythonruntime
import utilsnginx

def main():
    parser = argparse.ArgumentParser(description='Run PHP buildpacks.')
    parser.add_argument('--buildpack', required=True, help='The ID of the buildpack to run')
    parser.add_argument('--phase', required=True, choices=['detect', 'build'],
                      help='The phase to run: detect or build')
    args = parser.parse_args()

    # Dictionary mapping buildpack IDs to their respective functions
    buildpacks = {
        "google.php.appengine": {
            "detect": phpappengine.detect,
            "build": phpappengine.build
        },
        "google.php.cloudfunctions": {
            "detect": phpcloudfunctions.detect,
            "build": phpcloudfunctions.build
        },
        "google.php.composer": {
            "detect": phpcomposer.detect,
            "build": phpcomposer.build
        },
        "google.php.composer-gcp-build": {
            "detect": phpcomposergcpbuild.detect,
            "build": phpcomposergcpbuild.build
        },
        "google.php.composer-install": {
            "detect": phpcomposerinstall.detect,
            "build": phpcomposerinstall.build
        },
        "google.php.functions-framework": {
            "detect": phpfunctionsframework.detect,
            "build": phpfunctionsframework.build
        },
        "google.php.runtime": {
            "detect": phpruntime.detect,
            "build": phpruntime.build
        },
        "google.php.supervisor": {
            "detect": phpsupervisor.detect,
            "build": phpsupervisor.build
        },
        "google.php.webconfig": {
            "detect": phpwebconfig.detect,
            "build": phpwebconfig.build
        },
        "google.python.runtime": {
            "detect": pythonruntime.detect,
            "build": pythonruntime.build
        },
        "google.utils.nginx": {
            "detect": utilsnginx.detect,
            "build": utilsnginx.build
        }
    }

    if args.buildpack not in buildpacks:
        print(f"Buildpack {args.buildpack} not found.")
        return 1

    if args.phase == 'detect':
        buildpacks[args.buildpack]['detect']()
    elif args.phase == 'build':
        buildpacks[args.buildpack]['build']()

if __name__ == "__main__":
    main()
