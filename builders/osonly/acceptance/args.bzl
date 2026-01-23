"""Module for initializing arguments by osonly version"""

gcp_runtime_versions = [
    "osonly24",
]

osonly_gcp_runtime_versions = gcp_runtime_versions

version_to_builder_stack = {
    "osonly24": "google-24-full",
}

version_to_run_stack_min = {
    "osonly24": "google-24",
}
