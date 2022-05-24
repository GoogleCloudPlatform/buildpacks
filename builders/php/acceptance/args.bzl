load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by PHP version"""

def phpargs():
    """Create a new key-value map of arguments for php test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "7.2": php72args(),
        "7.3": php73args(),
        "7.4": php74args(),
        "8.1": php81args(),
    }
    return args

def php72args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/php72/run")

def php73args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/php73/run")

def php74args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/php74/run")

def php81args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/php81/run")

def newArgs(runImage):
    return {
        "-run-image-override": runImage,
    }
