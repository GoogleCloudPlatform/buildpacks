"""Module for initializing aruments by java version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# java8 is gen1 runtime so it's not using buildpacks
gae_runtime_versions = {n: n.replace("java", "") for n in gae_runtimes if n != "java8"}
gcf_runtime_versions = {n: n.replace("java", "") for n in gcf_runtimes}
flex_runtime_versions = {n: n.replace("java", "") for n in flex_runtimes}

# The union of all versions across all products.
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
java_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]
