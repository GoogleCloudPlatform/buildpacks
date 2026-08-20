"""
The runner module executes Python language builder buildpacks using FastAPI and Pydantic.
"""

import argparse
from typing import Dict, Any
from pydantic import BaseModel
from fastapi import FastAPI
import asyncio

app = FastAPI()

class Buildpack(BaseModel):
    detect: callable
    build: callable

# Import buildpack modules
from .appengine import lib as pythonappengine
from .functions_framework import lib as pythonfunctionsframework
from .functions_framework_compat import lib as pythonfunctionsframeworkcompat
from .link_runtime import lib as pythonlinkruntime
from .missing_entrypoint import lib as pythonmissingentrypoint
from .pip import lib as pythonpip
from .poetry import lib as pythonpoetry
from .runtime import lib as pythonruntime
from .uv import lib as pythonuv
from .webserver import lib as pythonwebserver

# Register buildpacks
buildpacks: Dict[str, Buildpack] = {
    "google.python.appengine": Buildpack(
        detect=pythonappengine.detect,
        build=pythonappengine.build
    ),
    "google.python.functions-framework": Buildpack(
        detect=pythonfunctionsframework.detect,
        build=pythonfunctionsframework.build
    ),
    "google.python.functions-framework-compat": Buildpack(
        detect=pythonfunctionsframeworkcompat.detect,
        build=pythonfunctionsframeworkcompat.build
    ),
    "google.python.link-runtime": Buildpack(
        detect=pythonlinkruntime.detect,
        build=pythonlinkruntime.build
    ),
    "google.python.missing-entrypoint": Buildpack(
        detect=pythonmissingentrypoint.detect,
        build=pythonmissingentrypoint.build
    ),
    "google.python.pip": Buildpack(
        detect=pythonpip.detect,
        build=pythonpip.build
    ),
    "google.python.poetry": Buildpack(
        detect=pythonpoetry.detect,
        build=pythonpoetry.build
    ),
    "google.python.runtime": Buildpack(
        detect=pythonruntime.detect,
        build=pythonruntime.build
    ),
    "google.python.webserver": Buildpack(
        detect=pythonwebserver.detect,
        build=pythonwebserver.build
    ),
    "google.python.uv": Buildpack(
        detect=pythonuv.detect,
        build=pythonuv.build
    )
}

async def run_buildpack(buildpack_id: str, phase: str) -> Dict[str, Any]:
    """
    Run the specified buildpack phase.

    Args:
        buildpack_id: The ID of the buildpack to run
        phase: The phase to execute ('detect' or 'build')

    Returns:
        dict: The result of the buildpack execution

    Raises:
        ValueError: If buildpack_id or phase is invalid
    """
    if buildpack_id not in buildpacks:
        raise ValueError(f"Buildpack {buildpack_id} not found")

    bp = buildpacks[buildpack_id]

    if phase == "detect":
        return await bp.detect()
    elif phase == "build":
        return await bp.build()
    else:
        raise ValueError(f"Invalid phase: {phase}")

def main():
    parser = argparse.ArgumentParser(description='Run Python buildpacks')
    parser.add_argument('--buildpack', type=str, required=True,
                       help='The ID of the buildpack to run (e.g., google.python.runtime)')
    parser.add_argument('--phase', type=str, required=True,
                       choices=['detect', 'build'],
                       help='The phase to run')

    args = parser.parse_args()

    result = asyncio.run(run_buildpack(args.buildpack, args.phase))
    print(result)

if __name__ == "__main__":
    main()
