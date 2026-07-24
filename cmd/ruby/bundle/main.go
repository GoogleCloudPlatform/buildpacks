from fastapi import FastAPI, status
from pydantic import BaseModel
import logging

# Initialize FastAPI app
app = FastAPI()

class BuildpackState(BaseModel):
    enabled: bool = True

async def get_state():
    return BuildpackState(enabled=True)

@app.post("/build")
async def build(state: BuildpackState = Depends(get_state)):
    if not state.enabled:
        return {"status": status.HTTP_204_NO_CONTENT, "message": "Buildpack disabled"}

    try:
        detected = await detect.detect()
        if not detected:
            return {"status": status.HTTP_404_NOT_FOUND, "message": "No Ruby files found"}

        build_result = await build.build_dependencies()
        return {"status": status.HTTP_200_OK, "result": build_result}
    except Exception as e:
        logging.error(f"Buildpack error: {str(e)}")
        raise

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
