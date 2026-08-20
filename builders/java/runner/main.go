import argparse
import logging
from typing import Dict, Any, Callable, Optional

from pydantic import BaseModel
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware

from google_buildpacks.cmd.java.appengine.lib import detect_fn as java_appengine_detect, build_fn as java_appengine_build
# ... (similar imports for all other Java buildpack functions)

app = FastAPI()

class BuildpackFuncs(BaseModel):
    detect: Callable
    build: Callable

buildpacks: Dict[str, BuildpackFuncs] = {
    "google.java.appengine": BuildpackFuncs(
        detect=java_appengine_detect,
        build=java_appengine_build
    ),
    # ... (similar setup for all other Java buildpack functions)
}

@app.get("/run-buildpack")
async def run_buildpack(request: Request, buildpack_id: str, phase: str):
    if buildpack_id not in buildpacks:
        raise HTTPException(status_code=404, detail="Buildpack not found")

    func_map = {
        'detect': buildpacks[buildpack_id].detect,
        'build': buildpacks[buildpack_id].build
    }

    if phase not in ['detect', 'build']:
        raise HTTPException(status_code=400, detail="Invalid phase")

    try:
        result = await func_map[phase]()
        return {"result": result}
    except Exception as e:
        logging.error(f"Error running {buildpack_id} {phase}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

def main():
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

if __name__ == "__main__":
    main()
