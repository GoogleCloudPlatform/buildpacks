import os
from google_cloud_platform.buildpacks import gcp_buildpacks as gcp

from .lib import detect as lib_detect, build as lib_build

class LinkRuntimeBuildpack:
    def detect(self):
        """Determine if this buildpack should be applied."""
        return lib_detect()

    def build(self):
        """Link the Python runtime to the GAE base image."""
        lib_build()

def main():
    """Main entry point for the link-runtime buildpack."""
    buildpack = LinkRuntimeBuildpack()
    if buildpack.detect():
        buildpack.build()

if __name__ == "__main__":
    main()
