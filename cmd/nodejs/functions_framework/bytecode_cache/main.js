import asyncio
import importlib.util
import json
import logging
import os
import shutil
from pathlib import Path
from typing import Dict, Any

import aiofiles
import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, validator

app = FastAPI()

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class InputData(BaseModel):
    entrypoint: str
    cache_dir_name: str

    @validator('entrypoint', 'cache_dir_name')
    def check_not_empty(cls, value):
        if not value.strip():
            raise ValueError("Field cannot be empty")
        return value

async def generate_bytecode_cache(data: InputData) -> Dict[str, Any]:
    try:
        entrypoint = data.entrypoint
        cache_dir_name = data.cache_dir_name

        # Get the current working directory
        cwd = Path.cwd()
        cache_path = cwd / cache_dir_name

        # Check if cache directory exists and remove it
        if os.path.exists(cache_path):
            await asyncio.to_thread(shutil.rmtree, str(cache_path), ignore_errors=True)

        # Create the cache directory
        cache_path.mkdir(parents=True, exist_ok=True)

        # Get module spec for entrypoint
        spec = importlib.util.find_spec(entrypoint)
        if not spec:
            raise ValueError(f"Module {entrypoint} not found.")

        # Read source code
        async with aiofiles.open(spec.origin, 'rb') as f:
            source_bytes = await f.read()

        # Compile the source into a code object
        code = compile(source_bytes, str(cache_path / (spec.name + '.py')), 'exec')

        # Write bytecode to cache directory
        target_pyc = cache_path / (spec.name + '.pyc')
        async with aiofiles.open(target_pyc, 'wb') as f:
            await f.write(importlib.util._write_bytecode(code))

        # Import the module to trigger other compilations
        importlib.import_module(entrypoint)

        return {"status": "success", "message": "Cache generation complete."}

    except Exception as e:
        logger.error(f"Error during cache generation: {e}")
        return {"status": "error", "message": str(e), "details": traceback.format_exc()}

@app.post('/generate')
async def handle_generate(data: InputData):
    result = await generate_bytecode_cache(data)
    if result['status'] == 'error':
        raise HTTPException(status_code=500, detail=result['message'])
    return result

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
