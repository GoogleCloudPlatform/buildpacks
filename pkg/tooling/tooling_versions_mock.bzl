"""Mock tooling versions for testing."""

MOCK_TOOLING_VERSIONS = {
    "python": {
        "default": {
            "uv": "0.11.0",
        },
        "runtimes": [
            {
                "names": [
                    "python39",
                ],
                "tools": {
                    "poetry": "2.2.1",
                },
            },
        ],
    },
    "nodejs": {
        "default": {
            "yarn": "1.22.22",
            "pnpm": "10.32.1",
        },
        "runtimes": [
            {
                "stacks": [
                    "ubuntu1804",
                ],
                "tools": {
                    "pnpm": "10.12.4",
                },
            },
        ],
    },
    "java": {
        "default": {
            "gradle": "9.4.1",
        },
        "runtimes": [
            {
                "names": [
                    "java11",
                ],
                "stacks": [
                    "ubuntu1804",
                ],
                "tools": {
                    "gradle": "8.14.3",
                },
            },
        ],
    },
}
