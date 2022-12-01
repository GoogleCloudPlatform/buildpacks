load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by PHP version"""

def phpargs(runImageTag = ""):
    """Create a new key-value map of arguments for php test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "7.2": newArgs("72", runImageTag),
        "7.3": newArgs("73", runImageTag),
        "7.4": newArgs("74", runImageTag),
        "8.1": newArgs("81", runImageTag),
    }
    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/php" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/php" + version + "/run"
