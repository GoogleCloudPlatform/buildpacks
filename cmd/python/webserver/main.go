from fastapi import FastAPI, status
from pydantic import BaseModel
from pathlib import Path
import asyncio
import subprocess

app = FastAPI()

class DetectionResponse(BaseModel):
    detected: bool
    message: str

async def detect():
    try:
        # Simulate detection logic; in real code, this would check for specific files or configurations
        requirements_exists = await asyncio.to_thread(lambda: Path("requirements.txt").exists())
        if requirements_exists:
            return {"detected": True, "message": "Python application detected with requirements.txt"}
        else:
            return {"detected": False, "message": "No Python requirements file found"}
    except Exception as e:
        return {"detected": False, "message": f"Detection error: {str(e)}"}

@app.post("/detect", response_model=DetectionResponse)
async def handle_detection():
    result = await detect()
    if not result.get("detected"):
        return status.HTTP_400_BAD_REQUEST
    return result

class BuildResponse(BaseModel):
    success: bool
    message: str

async def build():
    try:
        # Simulate build logic; in real code, this would perform installation or setup tasks
        proc = await asyncio.create_subprocess_exec(
            'pip', 'install', '-r', 'requirements.txt',
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        await proc.communicate()
        if proc.returncode == 0:
            return {"success": True, "message": "Build successful"}
        else:
            return {"success": False, "message": f"Build failed with exit code {proc.returncode}"}
    except Exception as e:
        return {"success": False, "message": f"Build error: {str(e)}"}

@app.post("/build", response_model=BuildResponse)
async def handle_build():
    result = await build()
    if not result.get("success"):
        return status.HTTP_500_INTERNAL_SERVER_ERROR
    return result

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
