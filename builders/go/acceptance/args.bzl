load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by GO version"""

def goargs(runImageTag):
    """Create a new key-value map of arguments for go test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "1.11": newArgs("111", runImageTag),
        "1.12": newArgs("112", runImageTag),
        "1.13": newArgs("113", runImageTag),
        "1.14": newArgs("114", runImageTag),
        "1.15": newArgs("115", runImageTag),
        "1.16": newArgs("116", runImageTag),
        "1.18": newArgs("118", runImageTag),
        "1.19": newArgs("119", runImageTag),
    }
    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": "gcr.io/gae-runtimes/buildpacks/go" + version + "/run:" + runImageTag,
    }
