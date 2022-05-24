load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by GO version"""

def goargs():
    """Create a new key-value map of arguments for go test

    Returns:
        A key-value map of the arguments
    """
    args = {
        "1.11": go111args(),
        "1.12": go112args(),
        "1.13": go113args(),
        "1.14": go114args(),
        "1.15": go115args(),
        "1.16": go116args(),
    }
    return args

def go111args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go111/run")

def go112args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go112/run")

def go113args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go113/run")

def go114args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go114/run")

def go115args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go115/run")

def go116args():
    return newArgs("gcr.io/gae-runtimes/buildpacks/go116/run")

def newArgs(runImage):
    return {
        "-run-image-override": runImage,
    }
