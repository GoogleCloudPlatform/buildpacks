load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by Ruby version"""

def rubyargs():
    """Create a new key-value map of arguments for Ruby acceptance tests

    Returns:
        A key-value map of the arguments
    """
    args = {
        "2.5.9": ruby25args(),
        "2.6.10": ruby26args(),
        "2.7.6": ruby27args(),
        "3.0.4": ruby30args(),
    }
    return args

def ruby25args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/ruby25/run")

def ruby26args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/ruby26/run")

def ruby27args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/ruby27/run")

def ruby30args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/ruby30/run")

def newArgs(runImage):
    return {
        "-run-image-override": runImage,
    }
