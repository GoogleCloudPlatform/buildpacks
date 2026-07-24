# The runner module executes buildpacks for the Universal builder.
import argparse

from builders.gcp.base.runner import gcp
from builders.cpp.clear_source.lib import detect as cppclearsource_detect, build as cppclearsource_build
from builders.cpp.functions_framework.lib import detect as cppfunctionsframework_detect, build as cppfunctionsframework_build
# ... (continue importing all required functions for each buildpack)
from builders.utils.nginx.lib import detect as utilsnginx_detect, build as utilsnginx_build
from builders.static.serve.lib import detect as staticserve_detect, build as staticserve_build

def main():
    parser = argparse.ArgumentParser(description='Run buildpacks')
    parser.add_argument('--buildpack', type=str, required=True,
                       help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', type=str, required=True,
                       help='The phase to run: "detect" or "build"')
    args = parser.parse_args()

    # Register buildpack functions here
    buildpacks = {
        "google.cpp.clear-source": {
            "detect": cppclearsource_detect,
            "build": cppclearsource_build
        },
        "google.cpp.functions-framework": {
            "detect": cppfunctionsframework_detect,
            "build": cppfunctionsframework_build
        },
        # ... (continue adding all buildpack entries)
        "google.utils.nginx": {
            "detect": utilsnginx_detect,
            "build": utilsnginx_build
        },
        "google.static.serve": {
            "detect": staticserve_detect,
            "build": staticserve_build
        }
    }

    gcp.main_runner(buildpacks, args.buildpack, args.phase)

if __name__ == "__main__":
    main()
