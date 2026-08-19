import json
import os
import re
import sys


def main():
    mock_process_map_env = "MOCKPROCESSUTIL_ENV_HELPER_MOCKPROCESSMAP"
    mocks_json = os.getenv(mock_process_map_env)
    if not mocks_json:
        print(f"Environment variable {mock_process_map_env} must be set.", file=sys.stderr)
        sys.exit(1)

    try:
        mocks = json.loads(mocks_json)
    except json.JSONDecodeError as e:
        print(f"Failed to parse JSON from environment variable: {e}", file=sys.stderr)
        sys.exit(1)

    full_command = ' '.join(sys.argv[1:])
    mock_match = None

    for pattern, config in mocks.items():
        compiled_pattern = re.compile(pattern)
        if compiled_pattern.search(full_command):
            mock_match = config
            break

    if not mock_match:
        sys.exit(0)

    stdout = mock_match.get("stdout", "")
    stderr = mock_match.get("stderr", "")
    exit_code = mock_match.get("exit_code", 0)

    if stdout:
        print(stdout, file=sys.stdout)

    if stderr:
        print(stderr, file=sys.stderr)

    sys.exit(exit_code)


if __name__ == "__main__":
    main()
