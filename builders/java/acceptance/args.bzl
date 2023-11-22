load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by java version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# java8 is gen1 runtime so it's not using buildpacks
gae_runtime_versions = {n: n.replace("java", "") for n in gae_runtimes if n != "java8"}
gcf_runtime_versions = {n: n.replace("java", "") for n in gcf_runtimes}
flex_runtime_versions = {n: n.replace("java", "") for n in flex_runtimes}

# The union of all versions across all products.
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)

def javaargs(runImageTag = ""):
    """Create a new key-value map of arguments for java test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for _n, version in gae_runtime_versions.items():
        args[version] = newArgs(version, runImageTag)

    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/java" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/java" + version + "/run"
