import asyncio
from typing import Optional

import typer
from pydantic import BaseModel

class BuildpackInfo(BaseModel):
    id: str
    version: str
    checksum: Optional[str] = None

async def main():
    from lib.detector import Detector
    from lib.builder import Builder

    detector = Detector()
    builder = Builder()

    try:
        stdin = await asyncio.get_event_loop().run_in_executor(None, open, '/dev/stdin', 'r')
        stdout = await asyncio.get_event_loop().run_in_executor(None, open, '/dev/stdout', 'w')

        detected = await detector.detect(stdin, stdout)
        if not detected:
            return

        buildpack_info = await builder.build(detected.version)

        print(f"Buildpack ID: {buildpack_info.id}")
        print(f"Version: {buildpack_info.version}")
        if buildpack_info.checksum:
            print(f"Checksum: {buildpack_info.checksum}")

    except Exception as e:
        print(f"Error processing request: {str(e)}")
        raise typer.Exit(1)

if __name__ == "__main__":
    asyncio.run(main())
