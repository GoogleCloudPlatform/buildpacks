load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by python version"""

def pythonargs(runImageTag = ""):
    """Create a new key-value map of arguments for python test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "3.7.12": newArgs("37", runImageTag),
        "3.8.12": newArgs("38", runImageTag),
        "3.9.10": newArgs("39", runImageTag),
        "3.10.4": newArgs("310", runImageTag),
        "3.11.0": newArgs("311", runImageTag),
    }
    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/python" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/python" + version + "/run"
