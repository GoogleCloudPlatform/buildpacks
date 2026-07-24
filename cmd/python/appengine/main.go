from fastapi import FastAPI
from pydantic import BaseModel
import logging
import os

app = FastAPI(title="App Engine Buildpack", version="1.0.0")

class DetectRequest(BaseModel):
    project_id: str
    runtime: str
    files: list[str]

class DetectResponse(BaseModel):
    applies: bool
    message: str | None

class BuildRequest(BaseModel):
    config: dict
    project_id: str

@app.post("/detect")
async def detect(request: DetectRequest) -> DetectResponse:
    """
    Detect if the appengine buildpack should be applied.

    Args:
        request (DetectRequest): Detection parameters

    Returns:
        DetectResponse: Whether the buildpack applies and optional message
    """
    # Check for App Engine files
    is_app_engine = os.path.exists("app.yaml") or os.path.exists("app.yml")
    return DetectResponse(applies=is_app_engine, message=None)

@app.post("/build")
async def build(request: BuildRequest) -> dict:
    """
    Perform the appengine build operation.

    Args:
        request (BuildRequest): Build configuration and project ID

    Returns:
        dict: Build result
    """
    logging.info("Running App Engine buildpack")

    # Convert env vars to JSON format
    env_vars = {
        "project_id": request.project_id,
        "runtime_config": request.config.get("runtime", {})
    }

    return {
        "env": env_vars,
        "processes": [
            {
                "command": ["python3", "-m", "google_cloud_platform.app_engine.serve"],
                "type": "web"
            }
        ]
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)
