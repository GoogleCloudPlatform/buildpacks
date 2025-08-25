"""
Generated bzl file. Do not update manually.
We run acceptance tests only on full stacks.
"""

gae_runtimes = {
    "python27": "2.7",
    "python37": "3.7.17",
    "python38": "3.8.20",
    "python39": "3.9.23",
    "python310": "3.10.18",
    "python311": "3.11.13",
    "python312": "3.12.11",
    "python313": "3.13.7",
}

gcf_runtimes = {
    "python37": "3.7.17",
    "python38": "3.8.20",
    "python39": "3.9.23",
    "python310": "3.10.18",
    "python311": "3.11.13",
    "python312": "3.12.11",
    "python313": "3.13.7",
}

flex_runtimes = {
    "python38": "3.8.20",
    "python39": "3.9.23",
    "python310": "3.10.18",
    "python311": "3.11.13",
    "python312": "3.12.11",
    "python313": "3.13.7",
}

version_to_stack = {
    "python27": "google-18-full",
    "python310": "google-22-full",
    "python311": "google-22-full",
    "python312": "google-22-full",
    "python313": "google-22-full",
    "python37": "google-18-full",
    "python38": "google-18-full",
    "python39": "google-18-full",
}
