load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing aruments by PHP version"""

load(":runtime.bzl", "gae_runtimes", "gcf_runtimes")

# php55 is gen1 runtime so excluding it from the list.
gae_php_runtime_versions = {n: n[3] + "." + n[4:] for n in gae_runtimes if n != "php55"}
gcf_php_runtime_versions = {n: n[3] + "." + n[4:] for n in gcf_runtimes}

# flex uses php 8+ with the same runtimes as gcf.
flex_php_runtime_versions = [v[3] + "." + v[4:] for v in gcf_runtimes if v != "php74"]

def phpargs(runImageTag = ""):
    """Create a new key-value map of arguments for php test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for _n, version in gae_php_runtime_versions.items():
        args[version] = newArgs(version.replace(".", ""), runImageTag)
    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/php" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/php" + version + "/run"
