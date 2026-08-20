from fastapi import FastAPI, Body
import asyncio
from lib import detect, build  # Assuming these are async functions

app = FastAPI()

@app.post("/detect")
async def detect_endpoint(data: dict = Body(...)):
    return await detect.detect_fn(data)

@app.post("/build")
async def build_endpoint(data: dict = Body(...)):
    return await build.build_fn(data)

async def main():
    # Run the server using uvicorn with asyncio
    import uvicorn
    config = uvicorn.Config(app, host="0.0.0.0", port=8000)
    server = uvicorn.Server(config=config)
    await server.serve()

if __name__ == "__main__":
    asyncio.run(main())
