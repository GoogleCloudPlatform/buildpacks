"""
Generated bzl file. Do not update manually.
We run acceptance tests only on full stacks.
"""

gae_runtimes = {
    "java8": "8.0",
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.2_10.0.LTS",
}

gcf_runtimes = {
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.2_10.0.LTS",
}

flex_runtimes = {
    "java11": "11.0",
    "java17": "17.0",
    "java21": "21.0",
    "java25": "25.0.2_10.0.LTS",
}

version_to_stack = {
    "java11": "google-18-full",
    "java17": "google-22-full",
    "java21": "google-22-full",
    "java25": "google-24-full",
    "java8": "google-18-full",
}
