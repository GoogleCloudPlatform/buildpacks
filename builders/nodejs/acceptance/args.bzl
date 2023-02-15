load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by nodejs version"""

def nodejsargs(runImageTag = ""):
    """Create a new key-value map of arguments for nodejs tests

    Returns:
        A key-value map of the arguments
    """
    args = {
        "8.17.0": newArgs("8", runImageTag),
        "10.24.1": newArgs("10", runImageTag),
        "12.22.12": newArgs("12", runImageTag),
        "14.18.3": newArgs("14", runImageTag),
        "16.13.2": newArgs("16", runImageTag),
        "18.10.0": newArgs("18", runImageTag),
        "19.6.0": newArgs("20", runImageTag),
    }
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
