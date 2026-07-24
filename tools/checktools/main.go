from typing import Optional
import logging
from fastapi import FastAPI
from pydantic import BaseModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI()

class ToolCheckResponse(BaseModel):
    status: str
    success: bool
    message: str
    error_details: Optional[str] = None

@app.get("/tools/installed", response_model=ToolCheckResponse)
async def check_installed_tools():
    try:
        logger.info("Checking tools")
        # Assuming we convert the original checktools.Installed() logic to Python
        # Replace this with actual implementation
        if await checktools.installed():
            return ToolCheckResponse(
                status="SUCCESS",
                success=True,
                message="All tools are correctly installed"
            )
    except Exception as e:
        logger.error(f"Error checking tools: {e}")
        return ToolCheckResponse(
            status="ERROR",
            success=False,
            message=str(e),
            error_details=str(e)
        )

@app.get("/pack/version", response_model=ToolCheckResponse)
async def check_pack_version():
    try:
        logger.info("Checking pack version")
        # Assuming we convert the original checktools.PackVersion() logic to Python
        # Replace this with actual implementation
        if await checktools.pack_version():
            return ToolCheckResponse(
                status="SUCCESS",
                success=True,
                message="Pack version is valid"
            )
    except Exception as e:
        logger.error(f"Error checking pack version: {e}")
        return ToolCheckResponse(
            status="ERROR",
            success=False,
            message=str(e),
            error_details=str(e)
        )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000, debug=True)
