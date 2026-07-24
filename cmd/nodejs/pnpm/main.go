from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
from typing import Dict, Any

# Replace with actual imports from your library
from .lib import detect_fn, build_fn

app = FastAPI()

class BuildpackRequest(BaseModel):
    """
    Request model for the pnpm buildpack endpoint.
    Attributes:
        app_dir: Application directory path
        env: Environment variables dictionary
        config: Buildpack configuration settings
    """
    app_dir: str
    env: Dict[str, str]
    config: Dict[str, Any]

class BuildpackResponse(BaseModel):
    """
    Response model for the pnpm buildpack endpoint.
    Attributes:
        status: Operation status (success or failure)
        message: Status message description
        output: Output details from the build operation
    """
    status: str
    message: str
    output: Dict[str, Any]

@app.post("/build")
async def handle_build(request_data: BuildpackRequest) -> BuildpackResponse:
    """
    Main endpoint handler for pnpm buildpack operations.
    Handles both detection and building of pnpm dependencies.

    Args:
        request_data: BuildpackRequest containing application directory,
                      environment variables, and configuration settings.

    Returns:
        BuildpackResponse with operation status and details.

    Raises:
        HTTPException: If any step in the build process fails.
    """
    try:
        # Run detection first
        detect_result = await asyncio.get_event_loop().run_in_executor(None, detect_fn, request_data.app_dir, request_data.env)

        if not detect_result['success']:
            raise HTTPException(status_code=400, detail=detect_result['message'])

        # Proceed with building if detected successfully
        build_result = await asyncio.get_event_loop().run_in_executor(None, build_fn, request_data.app_dir, request_data.env, request_data.config)

        return BuildpackResponse(
            status="success",
            message="pnpm build completed successfully",
            output=build_result
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

def main():
    """
    Main entry point for the FastAPI server.
    Starts the server with uvicorn on port 8000.
    """
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

if __name__ == "__main__":
    main()
