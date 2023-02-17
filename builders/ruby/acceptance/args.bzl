load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Module for initializing arguments by Ruby version"""

def rubyargs(runImageTag = ""):
    """Create a new key-value map of arguments for Ruby acceptance tests

    Returns:
        A key-value map of the arguments
    """
    args = {
        "2.5.9": newArgs("25", runImageTag),
        "2.6.10": newArgs("26", runImageTag),
        "2.7.6": newArgs("27", runImageTag),
        "3.0.4": newArgs("30", runImageTag),
        "3.2.0": newArgs("32", runImageTag),
    }
    return args

def newArgs(version, runImageTag):
    return {
        "-run-image-override": runImage(version, runImageTag),
    }

# If an image tag is specified, we get the run image from the 'gae-runtimes-private' repository.
# This can be used in Rapid pipelines and for local testing.
# If no 'runImageTag' is provided, we get the latest runtime image being used in PROD.
def runImage(version, runImageTag):
    if runImageTag != "":
        return "gcr.io/gae-runtimes-private/buildpacks/ruby" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/ruby" + version + "/run"
