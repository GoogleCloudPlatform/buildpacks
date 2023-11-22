load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by nodejs version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# nodejs8 is decommissioned (was never available on flex)
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if not v.startswith("8")}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if not v.startswith("8")}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)

def nodejsargs(runImageTag = ""):
    """Create a new key-value map of arguments for nodejs tests

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for n, v in gae_runtimes.items():
        args[v] = newArgs(n.replace("nodejs", ""), runImageTag)

    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/nodejs" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/nodejs" + version + "/run"
