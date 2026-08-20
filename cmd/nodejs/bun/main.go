from typing import Optional
import asyncio
import subprocess
from pydantic import BaseModel
import typer

# Data models for request/response
class DetectRequest(BaseModel):
    buildpack_id: str
    app_dir: str
    cache_dir: str
    output_dir: str

class DetectResponse(BaseModel):
    id: str
    version: str
    env_vars: dict[str, str]

class BuildRequest(BaseModel):
    app_dir: str
    cache_dir: str
    output_dir: str
    buildpack_plan: list[dict]

class BuildResponse(BaseModel):
    success: bool
    message: Optional[str] = None

async def detect(request: DetectRequest) -> DetectResponse:
    """Detect function to check if Bun is present."""
    bun_path = "bun"

    # Check if Bun is available in the environment
    try:
        proc = await asyncio.create_subprocess_exec(
            bun_path, "version",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE)
        await proc.communicate()

        if proc.returncode == 0:
            return DetectResponse(
                id=request.buildpack_id,
                version="1.0.0",
                env_vars={
                    "BUN_VERSION": "latest",
                    "NODE_ENV": "production"
                }
            )
        else:
            raise Exception("Bun not found")
    except FileNotFoundError:
        raise Exception("Bun executable not found in path")

async def build(request: BuildRequest) -> BuildResponse:
    """Build function to install dependencies using Bun."""
    try:
        proc = await asyncio.create_subprocess_exec(
            "bun", "install",
            cwd=request.app_dir,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE)

        stdout, stderr = await proc.communicate()

        if proc.returncode == 0:
            return BuildResponse(success=True, message="Dependencies installed successfully")
        else:
            error_msg = stderr.decode().strip() or "Unknown error"
            return BuildResponse(success=False, message=error_msg)
    except Exception as e:
        return BuildResponse(success=False, message=str(e))

class BunBuildpack:
    def __init__(self):
        self.app = typer.Typer()

    async def main(self):
        """Main entrypoint for the buildpack."""
        self.app.command()(self.detect_command)
        self.app.command()(self.build_command)

    async def detect_command(self, request: DetectRequest):
        try:
            response = await detect(request)
            print(response.json())
        except Exception as e:
            print(f"Detection failed: {str(e)}")

    async def build_command(self, request: BuildRequest):
        try:
            response = await build(request)
            print(response.json())
        except Exception as e:
            print(f"Build failed: {str(e)}")

if __name__ == "__main__":
    buildpack = BunBuildpack()
    asyncio.run(buildpack.main())
