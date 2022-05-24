load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by nodejs version"""

def nodejsargs():
    """Create a new key-value map of arguments for nodejs test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "8.17.0": nodejs8args(),
        "10.24.1": nodejs10args(),
        "12.22.9": nodejs12args(),
        "14.18.3": nodejs14args(),
        "16.13.2": nodejs16args(),
    }
    return args

def nodejs8args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/nodejs8/run")

def nodejs10args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/nodejs10/run")

def nodejs12args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/nodejs12/run")

def nodejs14args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/nodejs14/run")

def nodejs16args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/nodejs16/run")

def newArgs(runImage):
    return {
        "-run-image-override": runImage,
    }
