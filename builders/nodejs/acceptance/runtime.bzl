"""
Generated bzl file. Do not update manually.
We run acceptance tests only on full stacks.
"""

gae_runtimes = {
    "nodejs8": "8.17.0",
    "nodejs10": "10.24.1",
    "nodejs12": "12.22.12",
    "nodejs14": "14.21.3",
    "nodejs16": "16.20.2",
    "nodejs18": "18.20.8",
    "nodejs20": "20.19.5",
    "nodejs22": "22.19.0",
    "nodejs24": "24.8.0",
}

gcf_runtimes = {
    "nodejs8": "8.17.0",
    "nodejs10": "10.24.1",
    "nodejs12": "12.22.12",
    "nodejs14": "14.21.3",
    "nodejs16": "16.20.2",
    "nodejs18": "18.20.8",
    "nodejs20": "20.19.5",
    "nodejs22": "22.19.0",
    "nodejs24": "24.8.0",
}

flex_runtimes = {
    "nodejs18": "18.20.8",
    "nodejs20": "20.19.5",
    "nodejs22": "22.19.0",
    "nodejs24": "24.8.0",
}

version_to_stack = {
    "nodejs10": "google-18-full",
    "nodejs12": "google-18-full",
    "nodejs14": "google-18-full",
    "nodejs16": "google-18-full",
    "nodejs18": "google-22-full",
    "nodejs20": "google-22-full",
    "nodejs22": "google-22-full",
    "nodejs24": "google-24-full",
    "nodejs8": "google-18-full",
}
