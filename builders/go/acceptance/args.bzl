"""Module for initializing aruments by GO version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

# Note that the app.yamls in the test apps are still hardcoded to at least 1.18.
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items()}

# GCF Legacy Worker is only available and used for the "GCF go111" runtime version so it needs to
# be handled separately (and explicitly in BUILD).
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if n != "go111"}

# The union of all versions across all products.
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)

def goargs(runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for go test

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
