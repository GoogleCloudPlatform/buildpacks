import asyncio
import json
import os
from pathlib import Path
from typing import Dict, Optional

from pydantic import BaseModel

class BuildpackInfo(BaseModel):
    detected: bool = False
    version: Optional[str] = None
    python_version: Optional[str] = None

async def detect(build_dir: str) -> Dict:
    """Detect if the Python runtime should be applied."""
    info = BuildpackInfo()

    # Check for requirements.txt or setup.py
    loop = asyncio.get_event_loop()
    files = await loop.run_in_executor(None, lambda:
        [f.name for f in Path(build_dir).iterdir() if f.is_file()]
    )

    if 'requirements.txt' in files or 'setup.py' in files:
        info.detected = True

        # Determine Python version
        python_version = os.environ.get("PYTHON_VERSION", "latest")
        info.python_version = python_version

    return json.loads(info.json())

async def build(build_dir: str) -> Dict:
    """Install the Python runtime."""
    info = BuildpackInfo()

    # Get detected info from environment or file
    try:
        with open(os.path.join(build_dir, '.python-buildpack'), 'r') as f:
            detected_info = json.load(f)
    except FileNotFoundError:
        detected_info = {}

    if not detected_info.get('detected'):
        return {'error': 'Python runtime not detected'}

    # Install Python version
    version = detected_info.get('version', os.environ.get("PYTHON_VERSION", "latest"))
    await asyncio.create_subprocess_exec(
        'python3',
        '-m', 'ensurepath', f'python/{version}',
        cwd=build_dir,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )

    info.version = version
    return json.loads(info.json())

async def main():
    buildpack_info = await detect(os.environ.get('BP_BUILDPACK_DIR', './'))
    if not buildpack_info.get('detected'):
        print(json.dumps({'error': 'Not a Python application'}))
        return

    try:
        result = await build(os.environ.get('BP_APP_DIR', './'))
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({'error': str(e)}))

if __name__ == "__main__":
    asyncio.run(main())
