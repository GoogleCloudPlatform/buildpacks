"""Module for initializing aruments by PHP version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# flex uses php 8+ with the same runtimes as gcf.
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}

# php55 is gen1 runtime so excluding it from the list.
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if n != "php55"}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items()}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)

def phpargs(runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for php test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for runtime, version in gae_runtime_versions.items():
        args[version] = newArgs(runtime, runImageTag, stack)
    return args

def newArgs(runtime, runImageTag, stack):
    return {
        "-run-image-override": runImage(runtime, runImageTag, stack),
    }

def runImage(runtime, runImageTag, stack):
    if stack != "":
        if runImageTag != "":
            return "us-docker.pkg.dev/gae-runtimes-private/gcp/" + stack + "/runtimes/" + runtime + ":" + runImageTag
        else:
            return "gcr.io/gae-runtimes/buildpacks/" + runtime + "/run"

    if runImageTag != "":
        return "us-docker.pkg.dev/gae-runtimes-private/gcp/buildpacks/" + runtime + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/" + runtime + "/run"
