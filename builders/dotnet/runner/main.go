# The runner executes buildpacks for the .NET language builder.

import argparse

# Import buildpack functions
from builders.dotnet.runner.appengine.lib import DetectFn as dotnetappengine_detect_fn, BuildFn as dotnetappengine_build_fn
from builders.dotnet.runner.appengine_main.lib import DetectFn as dotnetappenginemain_detect_fn, BuildFn as dotnetappenginemain_build_fn
from builders.dotnet.runner.flex.lib import DetectFn as dotnetflex_detect_fn, BuildFn as dotnetflex_build_fn
from builders.dotnet.runner.functions_framework.lib import DetectFn as dotnetfunctionsframework_detect_fn, BuildFn as dotnetfunctionsframework_build_fn
from builders.dotnet.runner.publish.lib import DetectFn as dotnetpublish_detect_fn, BuildFn as dotnetpublish_build_fn
from builders.dotnet.runner.runtime.lib import DetectFn as dotnetruntime_detect_fn, BuildFn as dotnetruntime_build_fn
from builders.dotnet.runner.sdk.lib import DetectFn as dotnetsdk_detect_fn, BuildFn as dotnetsdk_build_fn

# Register buildpack functions
buildpacks = {
    "google.dotnet.appengine": {
        "detect": dotnetappengine_detect_fn,
        "build": dotnetappengine_build_fn
    },
    "google.dotnet.appengine-main": {
        "detect": dotnetappenginemain_detect_fn,
        "build": dotnetappenginemain_build_fn
    },
    "google.dotnet.flex": {
        "detect": dotnetflex_detect_fn,
        "build": dotnetflex_build_fn
    },
    "google.dotnet.runtime": {
        "detect": dotnetruntime_detect_fn,
        "build": dotnetruntime_build_fn
    },
    "google.dotnet.sdk": {
        "detect": dotnetsdk_detect_fn,
        "build": dotnetsdk_build_fn
    },
    "google.dotnet.publish": {
        "detect": dotnetpublish_detect_fn,
        "build": dotnetpublish_build_fn
    },
    "google.dotnet.functions-framework": {
        "detect": dotnetfunctionsframework_detect_fn,
        "build": dotnetfunctionsframework_build_fn
    }
}

def main():
    parser = argparse.ArgumentParser(description='Run .NET buildpacks.')
    parser.add_argument('--buildpack', type=str, required=True)
    parser.add_argument('--phase', type=str, required=True)

    args = parser.parse_args()

    if args.buildpack not in buildpacks:
        print(f"Buildpack {args.buildpack} not found.")
        return 1

    bp = buildpacks[args.buildpack]

    if args.phase == 'detect':
        bp['detect']()
    elif args.phase == 'build':
        bp['build']()
    else:
        print("Invalid phase. Must be 'detect' or 'build'.")
        return 1

if __name__ == "__main__":
    main()
