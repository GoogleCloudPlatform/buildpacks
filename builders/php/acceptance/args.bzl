load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by PHP version"""

load(":runtime.bzl", "gae_runtimes", "gcf_runtimes")

# php55 is gen1 runtime so excluding it from the list.
gae_php_runtime_versions = {n: gae_runtimes[n] for n in gae_runtimes if n != "php55"}
gcf_php_runtime_versions = {n: gcf_runtimes[n] for n in gcf_runtimes}

# flex uses php 8+ with the same runtimes as gcf.
flex_php_runtime_versions = {n: gcf_runtimes[n] for n in gcf_runtimes if n != "php74"}

def phpargs(runImageTag = ""):
    """Create a new key-value map of arguments for php test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for runtime, version in gae_php_runtime_versions.items():
        args[version] = newArgs(runtime, runImageTag)
    return args

def newArgs(runtime, runImageTag):
    return {
        "-run-image-override": runImage(runtime, runImageTag),
    }

def runImage(runtime, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/" + runtime + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/" + runtime + "/run"
