load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by java version"""

def javaargs(runImageTag = ""):
    """Create a new key-value map of arguments for java test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "8": newArgs("8", runImageTag),
        "11": newArgs("11", runImageTag),
        "17": newArgs("17", runImageTag),
    }
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
