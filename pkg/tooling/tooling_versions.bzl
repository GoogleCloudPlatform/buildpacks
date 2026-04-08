"""
Tooling versions generated from tooling.textproto.
"""

TOOLING_VERSIONS = {
    "python": {
        "default": {
            "uv": "0.11.3",
            "poetry": "2.3.3",
        },
        "runtimes": [
            {
                "names": [
                    "python39",
                ],
                "stacks": [
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
            "pnpm": "10.33.0",
            "bun": "1.3.11",
        },
        "runtimes": [
            {
                "names": [
                ],
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
