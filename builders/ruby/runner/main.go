from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import asyncio
from typing import Dict, Any
import gcp_buildpacks  # Assuming equivalent Python package structure

app = FastAPI()

class BuildpackRequest(BaseModel):
    buildpack_id: str
    phase: str

# Register buildpack functions
buildpacks: Dict[str, Dict[str, callable]] = {}

def register_buildpack(buildpack_id: str, detect_fn, build_fn) -> None:
    buildpacks[buildpack_id] = {
        'detect': detect_fn,
        'build': build_fn
    }

@app.post("/run")
async def run_buildpack(request_data: BuildpackRequest) -> Dict[str, Any]:
    try:
        # Parse request data
        buildpack_id = request_data.buildpack_id
        phase = request_data.phase

        if not buildpack_id or not phase:
            raise HTTPException(
                status_code=400,
                detail="Both buildpack_id and phase must be provided"
            )

        # Run the buildpack based on phase
        if phase.lower() == 'detect':
            result = await asyncio.get_event_loop().run_in_executor(
                None,
                buildpacks[buildpack_id]['detect']
            )
        elif phase.lower() == 'build':
            result = await asyncio.get_event_loop().run_in_executor(
                None,
                buildpacks[buildpack_id]['build']
            )
        else:
            raise HTTPException(
                status_code=400,
                detail="Invalid phase. Must be either 'detect' or 'build'"
            )

        return {"result": result}
    except KeyError:
        raise HTTPException(
            status_code=404,
            detail=f"Buildpack '{buildpack_id}' not found"
        )
    except Exception as e:
        raise HTTPException(
            status_code=500,
            detail=str(e)
        )

def main():
    # Register buildpacks (equivalent to Go init() function)
    register_buildpack("google.nodejs.runtime", gcp_buildpacks.nodejsruntime.detect, gcp_buildpacks.nodejsruntime.build)
    register_buildpack("google.nodejs.yarn", gcp_buildpacks.nodejyarn.detect, gcp_buildpacks.nodejyarn.build)
    register_buildpack("google.ruby.appengine", gcp_buildpacks.rubyappengine.detect, gcp_buildpacks.rubyappengine.build)
    # ... (register all other buildpacks similarly)

if __name__ == "__main__":
    main()
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
