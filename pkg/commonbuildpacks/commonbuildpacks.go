"""
Package commonbuildpacks provides a function to get a map of common buildpacks.
"""

from typing import Dict
from pkg.gcpbuildpack import BuildpackFuncs
from cmd.config.entrypoint.lib import detect_fn as entrypoint_detect, build_fn as entrypoint_build
from cmd.config.flex.lib import detect_fn as flex_detect, build_fn as flex_build
from cmd.utils.archive_source.lib import detect_fn as archive_detect, build_fn as archive_build
from cmd.utils.label.lib import detect_fn as label_detect, build_fn as label_build


def common_buildpacks() -> Dict[str, BuildpackFuncs]:
    """
    Returns a dictionary of common buildpacks that are used by all language runtimes.
    """
    return {
        "google.config.entrypoint": BuildpackFuncs(
            detect=entrypoint_detect,
            build=entrypoint_build,
        ),
        "google.config.flex": BuildpackFuncs(
            detect=flex_detect,
            build=flex_build,
        ),
        "google.utils.archive-source": BuildpackFuncs(
            detect=archive_detect,
            build=archive_build,
        ),
        "google.utils.label-image": BuildpackFuncs(
            detect=label_detect,
            build=label_build,
        ),
    }
