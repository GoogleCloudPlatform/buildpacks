import json
import os
import re
import sys
from pydantic import BaseModel
from typing import Dict, Optional
import asyncio


class MockProcessConfig(BaseModel):
    stdout: str = ""
    stderr: str = ""
    exit_code: int = 0


class MockProcesses(BaseModel):
    __root__: Dict[str, MockProcessConfig]

async def main():
    mock_processes_json = os.getenv("MOCKPROCESSES_JSON")
    if not mock_processes_json:
        sys.stderr.write("MOCKPROCESSES_JSON environment variable must be set\n")
        sys.exit(1)

    try:
        mock_processes_dict = json.loads(mock_processes_json)
        mock_processes = MockProcesses(__root__=mock_processes_dict).__root__
    except json.JSONDecodeError as e:
        sys.stderr.write(f"Failed to parse MOCKPROCESSES_JSON: {e}\n")
        sys.exit(1)
    except Exception as e:
        sys.stderr.write(f"Invalid MOCKPROCESSES_JSON format: {e}\n")
        sys.exit(1)

    full_command = ' '.join(sys.argv[1:])

    matched_mock = None
    for command_regex, mock_config in mock_processes.items():
        pattern = re.compile(command_regex)
        if pattern.match(full_command):
            matched_mock = mock_config
            break

    if not matched_mock:
        sys.exit(0)

    loop = asyncio.get_event_loop()
    if matched_mock.stdout:
        await loop.run_in_executor(None, lambda: sys.stdout.write(matched_mock.stdout))
        await loop.run_in_executor(None, lambda: sys.stdout.flush())

    if matched_mock.stderr:
        await loop.run_in_executor(None, lambda: sys.stderr.write(matched_mock.stderr))
        await loop.run_in_executor(None, lambda: sys.stderr.flush())

    sys.exit(matched_mock.exit_code)

asyncio.run(main())
