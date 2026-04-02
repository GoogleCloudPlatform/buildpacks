"""
Tooling versions generated from tooling.textproto.
"""

TOOLING_VERSIONS = {
    "python": {
        "default": {
            "uv": "0.11.2",
            "poetry": "2.3.3",
        },
        "runtimes": [
        ],
    },
    "nodejs": {
        "default": {
            "yarn": "1.22.22",
            "pnpm": "10.33.0",
            "bun": "1.3.11",
        },
        "runtimes": [
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
                    "google.gae.18",
                    "google.18",
                ],
                "tools": {
                    "gradle": "8.14.3",
                },
            },
        ],
    },
}
