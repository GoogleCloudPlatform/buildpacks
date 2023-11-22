load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by dotnet version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items()}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if n != "dotnet"}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)

# We wanted to support dotnet 7 if someone in the OSS community wanted to build against it,
# but we also don't want to target dotnet 7 explicitly -- it is not a LTS release, only STS.
gcp_runtime_versions["dotnet7"] = "7.0.100"
