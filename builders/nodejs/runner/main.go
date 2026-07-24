from typing import Dict, Any
import argparse
import asyncio
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, BaseSettings

class BuildpackFuncs(BaseModel):
    detect: callable
    build: callable

class Settings(BaseSettings):
    buildpack_id: str = ""
    phase: str = ""

    class Config:
        env_file = ".env"

settings = Settings()

app = FastAPI()
router = app.router

# Register buildpack functions here
buildpacks: Dict[str, BuildpackFuncs] = {}

def init_buildpacks():
    global buildpacks

    # firebasebundle
    from firebase.bundle import lib as firebasebundle_lib
    buildpacks["google.firebase.firebasebundle"] = BuildpackFuncs(
        detect=firebasebundle_lib.DetectFn,
        build=firebasebundle_lib.BuildFn
    )

    # nodejs appengine
    from nodejs.appengine import lib as nodejsappengine_lib
    buildpacks["google.nodejs.appengine"] = BuildpackFuncs(
        detect=nodejsappengine_lib.DetectFn,
        build=nodejsappengine_lib.BuildFn
    )

    # nodejs firebaseangular
    from nodejs.firebaseangular import lib as nodejsfirebaseangular_lib
    buildpacks["google.nodejs.firebaseangular"] = BuildpackFuncs(
        detect=nodejsfirebaseangular_lib.DetectFn,
        build=nodejsfirebaseangular_lib.BuildFn
    )

    # nodejs firebasebundle
    from nodejs.firebasebundle import lib as nodejsfirebasebundle_lib
    buildpacks["google.nodejs.firebasebundle"] = BuildpackFuncs(
        detect=nodejsfirebasebundle_lib.DetectFn,
        build=nodejsfirebasebundle_lib.BuildFn
    )

    # nodejs firebasenextjs
    from nodejs.firebasenextjs import lib as nodejsfirebasenextjs_lib
    buildpacks["google.nodejs.firebasenextjs"] = BuildpackFuncs(
        detect=nodejsfirebasenextjs_lib.DetectFn,
        build=nodejsfirebasenextjs_lib.BuildFn
    )

    # nodejs firebasenx
    from nodejs.firebasenx import lib as nodejsfirebasenx_lib
    buildpacks["google.nodejs.firebasenx"] = BuildpackFuncs(
        detect=nodejsfirebasenx_lib.DetectFn,
        build=nodejsfirebasenx_lib.BuildFn
    )

    # nodejs functions_framework
    from nodejs.functions_framework import lib as nodejsfunctionsframework_lib
    buildpacks["google.nodejs.functions-framework"] = BuildpackFuncs(
        detect=nodejsfunctionsframework_lib.DetectFn,
        build=nodejsfunctionsframework_lib.BuildFn
    )

    # nodejs legacy_worker
    from nodejs.legacy_worker import lib as nodejslegacyworker_lib
    buildpacks["google.nodejs.legacy-worker"] = BuildpackFuncs(
        detect=nodejslegacyworker_lib.DetectFn,
        build=nodejslegacyworker_lib.BuildFn
    )

    # nodejs npm
    from nodejs.npm import lib as nodejsnpm_lib
    buildpacks["google.nodejs.npm"] = BuildpackFuncs(
        detect=nodejsnpm_lib.DetectFn,
        build=nodejsnpm_lib.BuildFn
    )

    # nodejs pnpm
    from nodejs.pnpm import lib as nodejspnpm_lib
    buildpacks["google.nodejs.pnpm"] = BuildpackFuncs(
        detect=nodejspnpm_lib.DetectFn,
        build=nodejspnpm_lib.BuildFn
    )

    # nodejs runtime
    from nodejs.runtime import lib as nodejsruntime_lib
    buildpacks["google.nodejs.runtime"] = BuildpackFuncs(
        detect=nodejsruntime_lib.DetectFn,
        build=nodejsruntime_lib.BuildFn
    )

    # nodejs turborepo
    from nodejs.turborepo import lib as nodejsturborepo_lib
    buildpacks["google.nodejs.turborepo"] = BuildpackFuncs(
        detect=nodejsturborepo_lib.DetectFn,
        build=nodejsturborepo_lib.BuildFn
    )

    # nodejs yarn
    from nodejs.yarn import lib as nodejsyarn_lib
    buildpacks["google.nodejs.yarn"] = BuildpackFuncs(
        detect=nodejsyarn_lib.DetectFn,
        build=nodejsyarn_lib.BuildFn
    )

    # nodejs bun
    from nodejs.bun import lib as nodejsbun_lib
    buildpacks["google.nodejs.bun"] = BuildpackFuncs(
        detect=nodejsbun_lib.DetectFn,
        build=nodejsbun_lib.BuildFn
    )

@app.get("/")
async def root():
    if not settings.buildpack_id or not settings.phase:
        raise HTTPException(status_code=400, detail="Missing required parameters")

    if settings.phase not in ["detect", "build"]:
        raise HTTPException(status_code=400, detail="Invalid phase value. Must be 'detect' or 'build'")

    if settings.buildpack_id not in buildpacks:
        raise HTTPException(status_code=404, detail=f"Buildpack {settings.buildpack_id} not found")

    func = buildpacks[settings.buildpack_id]
    if settings.phase == "detect":
        result = await asyncio.to_thread(func.detect)
    else:
        result = await asyncio.to_thread(func.build)

    return {"result": result}

def main():
    init_buildpacks()
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Node.js Buildpack Runner')
    parser.add_argument('--buildpack', type=str, help='The ID of the buildpack to run (e.g., google.nodejs.runtime)')
    parser.add_argument('--phase', type=str, help='The phase to run: "detect" or "build"')
    args = parser.parse_args()

    if args.buildpack:
        settings.buildpack_id = args.buildpack
    if args.phase:
        settings.phase = args.phase

    main()
