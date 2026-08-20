from fastapi import FastAPI
from pydantic import BaseModel
import asyncio

app = FastAPI()

class DetectionRequest(BaseModel):
    # Define fields as needed by DetectFn
    pass

class BuildRequest(BaseModel):
    # Define fields as needed by BuildFn
    pass

@app.post("/detect")
async def detect(request: DetectionRequest):
    # Call the detection function async
    result = await lib.detect_fn(request)
    return {"result": result}

@app.post("/build")
async def build(request: BuildRequest):
    # Call the build function async
    result = await lib.build_fn(request)
    return {"result": result}

if __name__ == "__main__":
    asyncio.run(app.run())
