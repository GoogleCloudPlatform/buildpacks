from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
import json
import logging
import os

app = FastAPI()

class Buildpack(BaseModel):
    project_id: str
    service_account: str
    runtime: str
    environment: dict
    files: dict

async def detect() -> bool:
    """Detect if npm is required based on presence of package.json."""
    try:
        # Asynchronous file check using asyncio's loop.run_in_executor
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(None, os.path.isfile, 'package.json')
        return result
    except Exception as e:
        logging.error(f"Error detecting npm: {e}")
        raise HTTPException(status_code=500, detail=str(e))

async def build(buildpack_data: Buildpack) -> dict:
    """Install npm dependencies asynchronously."""
    try:
        # Asynchronous command execution using asyncio's subprocess
        proc = await asyncio.create_subprocess_exec(
            'npm', 'install',
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE)

        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise HTTPException(status_code=400, detail=f"Error installing npm packages: {stderr.decode()}")

        return {"status": "success", "message": "Dependencies installed successfully"}
    except Exception as e:
        logging.error(f"Error building npm dependencies: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/build")
async def handle_build(buildpack_data: Buildpack) -> dict:
    """Handle the build request."""
    try:
        # Simulate detection and build process
        needs_npm = await detect()
        if not needs_npm:
            return {"status": "success", "message": "No npm dependencies required"}

        result = await build(buildpack_data)
        return result

    except HTTPException as e:
        raise e
    except Exception as e:
        logging.error(f"Unexpected error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

async def main():
    """Main entry point for the FastAPI application."""
    try:
        # Run the FastAPI server using asyncio's event loop
        config = uvicorn.Config(app, host="0.0.0.0", port=8000)
        server = uvicorn.Server(config)
        await server.serve()
    except Exception as e:
        logging.error(f"Error starting server: {e}")
        raise

if __name__ == "__main__":
    asyncio.run(main())
