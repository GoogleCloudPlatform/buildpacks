"""Module for initializing aruments by PHP version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# flex uses php 8+ with the same runtimes as gcf.
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}

# php55 is gen1 runtime so excluding it from the list.
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if n != "php55"}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items()}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
php_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]
