load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by GO version"""

load(":runtime.bzl", "gae_runtimes", "gcf_runtimes")

gae_go_runtime_versions = {n: gae_runtimes[n] for n in gae_runtimes}

# GCF Legacy Worker is only available and used for the "GCF go111" runtime version so it needs to
# be handled separately.
gcf_go_runtime_versions = {n: gcf_runtimes[n] for n in gcf_runtimes if n != "go111"}

def goargs(runImageTag = ""):
    """Create a new key-value map of arguments for go test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for runtime, version in gae_go_runtime_versions.items():
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
