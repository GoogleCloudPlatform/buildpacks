from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import sys
from typing import Dict, Any

# Define the FastAPI app
app = FastAPI()

# Import buildpack functions
from builders.php.appengine.lib import detect as php_appengine_detect, build as php_appengine_build
from builders.php.composer.lib import detect as php_composer_detect, build as php_composer_build
from builders.php.composer_gcp_build.lib import detect as php_composergcpbuild_detect, build as php_composergcpbuild_build
from builders.php.composer_install.lib import detect as php_composerinstall_detect, build as php_composerinstall_build
from builders.php.functions_framework.lib import detect as php_functionsframework_detect, build as php_functionsframework_build
from builders.php.runtime.lib import detect as php_runtime_detect, build as php_runtime_build
from builders.php.supervisor.lib import detect as php_supervisor_detect, build as php_supervisor_build
from builders.php.webconfig.lib import detect as php_webconfig_detect, build as php_webconfig_build
from builders.python.runtime.lib import detect as python_runtime_detect, build as python_runtime_build
from builders.utils.nginx.lib import detect as utils_nginx_detect, build as utils_nginx_build

# Register buildpack functions in a dictionary
buildpacks: Dict[str, Dict[str, Any]] = {
    "google.php.appengine": {
        "detect": php_appengine_detect,
        "build": php_appengine_build
    },
    "google.php.cloudfunctions": {
        "detect": php_appengine_detect,  # Placeholder; actual function should be imported
        "build": php_appengine_build     # Placeholder; actual function should be imported
    },
    "google.php.composer": {
        "detect": php_composer_detect,
        "build": php_composer_build
    },
    "google.php.composer-gcp-build": {
        "detect": php_composergcpbuild_detect,
        "build": php_composergcpbuild_build
    },
    "google.php.composer-install": {
        "detect": php_composerinstall_detect,
        "build": php_composerinstall_build
    },
    "google.php.functions-framework": {
        "detect": php_functionsframework_detect,
        "build": php_functionsframework_build
    },
    "google.php.runtime": {
        "detect": php_runtime_detect,
        "build": php_runtime_build
    },
    "google.php.supervisor": {
        "detect": php_supervisor_detect,
        "build": php_supervisor_build
    },
    "google.php.webconfig": {
        "detect": php_webconfig_detect,
        "build": php_webconfig_build
    },
    "google.python.runtime": {
        "detect": python_runtime_detect,
        "build": python_runtime_build
    },
    "google.utils.nginx": {
        "detect": utils_nginx_detect,
        "build": utils_nginx_build
    }
}

# Pydantic model for request data validation
class RunRequest(BaseModel):
    buildpack_id: str
    phase: str

@app.post("/run")
async def run_buildpack(request_data: RunRequest):
    """
    Execute a specific buildpack's detect or build phase.

    Args:
        request_data (RunRequest): Contains buildpack ID and phase to execute.

    Returns:
        Dict[str, Any]: Result of the executed phase.

    Raises:
        HTTPException: If buildpack not found or invalid phase provided.
    """
    # Validate phase
    if request_data.phase.lower() not in ['detect', 'build']:
        raise HTTPException(
            status_code=422,
            detail="Invalid phase. Must be either 'detect' or 'build'."
        )

    # Get the buildpack functions
    buildpack = buildpacks.get(request_data.buildpack_id)
    if not buildpack:
        raise HTTPException(
            status_code=404,
            detail=f"Buildpack {request_data.buildpack_id} not found."
        )

    # Select appropriate function based on phase
    func = buildpack.get(request_data.phase.lower())
    if not func:
        raise HTTPException(
            status_code=501,
            detail=f"No {request_data.phase} function defined for this buildpack."
        )

    try:
        # Run the function asynchronously, assuming it's blocking
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, func)
        return {"result": result}
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=str(e)
        )
