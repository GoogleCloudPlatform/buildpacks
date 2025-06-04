"""Module for initializing arguments by nodejs version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# nodejs8 is decommissioned (was never available on flex)
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if not v.startswith("8")}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if not v.startswith("8")}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
nodejs_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]
