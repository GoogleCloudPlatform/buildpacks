"""
Generated bzl file. Do not update manually.
We run acceptance tests only on full stacks.
"""

gae_runtimes = {
    "java8": "8.0",
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.5_8.0.LTS.cve-2026-114938",
}

gcf_runtimes = {
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.5_8.0.LTS.cve-2026-114938",
}

flex_runtimes = {
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.5_8.0.LTS.cve-2026-114938",
}

version_to_stack = {
    "java11": "google-18-full",
    "java17": "google-22-full",
    "java21": "google-22-full",
    "java25": "google-24-full",
    "java8": "google-18-full",
}
