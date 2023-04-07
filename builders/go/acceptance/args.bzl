load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by GO version"""

load(":runtime.bzl", "gae_runtimes", "gcf_runtimes")

gae_go_runtime_versions = [v[2] + "." + v[3:] for v in gae_runtimes if v.startswith("go")]

# GCF Legacy Worker is only available and used for the "GCF go111" runtime version so it needs to
# be handled separately.
gcf_go_runtime_versions = [v[2] + "." + v[3:] for v in gcf_runtimes if v.startswith("go") and v != "go111"]

def goargs(runImageTag = ""):
    """Create a new key-value map of arguments for go test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for version in gae_go_runtime_versions:
        args[version] = newArgs(version.replace(".", ""), runImageTag)

    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/go" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/go" + version + "/run"
