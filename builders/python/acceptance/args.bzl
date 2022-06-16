load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by python version"""

def pythonargs():
    """Create a new key-value map of arguments for python test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "3.7.12": python37args(),
        "3.8.12": python38args(),
        "3.9.10": python39args(),
        "3.10.4": python310args(),
    }
    return args

def python37args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/python37/run")

def python38args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/python38/run")

def python39args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/python39/run")

def python310args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/python310/run")

def newArgs(runImage):
    return {
        "-run-image-override": runImage,
    }
