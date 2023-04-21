load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by nodejs version"""

load(":runtime.bzl", "gae_runtimes", "gcf_runtimes")

gae_nodejs_runtime_versions = [v for n, v in gae_runtimes.items()]
gcf_nodejs_runtime_versions = [v for n, v in gcf_runtimes.items()]

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
