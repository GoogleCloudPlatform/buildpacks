"""Module for initializing aruments by GO version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# Note that the app.yamls in the test apps are still hardcoded to at least 1.18.
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items()}

# GCF Legacy Worker is only available and used for the "GCF go111" runtime version so it needs to
# be handled separately (and explicitly in BUILD).
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if n != "go111"}

# The union of all versions across all products.
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
go_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]
