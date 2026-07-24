"""
Tooling versions generated from tooling.textproto.
"""

TOOLING_VERSIONS = {
    "python": {
        "default": {
            "uv": "0.11.31",
            "poetry": "2.4.1",
            "setuptools": "83.0.0",
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
            {
                "names": [
                    "python310",
                    "python311",
                    "python312",
                ],
                "stacks": [
                ],
                "tools": {
                    "setuptools": "81.0.0",
                },
            },
        ],
    },
    "nodejs": {
        "default": {
            "yarn": "1.22.22",
            "pnpm": "11.16.0",
            "bun": "1.3.14",
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
            {
                "names": [
                    "nodejs12",
                    "nodejs14",
                    "nodejs16",
                    "nodejs18",
                    "nodejs20",
                ],
                "stacks": [
                ],
                "tools": {
                    "pnpm": "10.12.4",
                },
            },
        ],
    },
    "java": {
        "default": {
            "maven": "3.9.16",
            "gradle": "9.6.1",
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
