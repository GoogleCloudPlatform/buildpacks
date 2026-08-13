"""Module for initializing arguments by static runtime version"""

gcp_runtime_versions = [
    "static24",
]

static_gcp_runtime_versions = gcp_runtime_versions

version_to_builder_stack = {
    "static24": "google-24-full",
}

version_to_run_stack_min = {
    "static24": "google-24",
}
