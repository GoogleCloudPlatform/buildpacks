"""Module for initializing aruments by python version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes")

flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if n != "python27"}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items()}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
python_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]

def pythonargs(runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for python test

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for n, v in gae_runtimes.items():
        args[v] = newArgs(n.replace("python", ""), runImageTag, stack)
    return args

def newArgs(version, runImageTag, stack):
    return {
        "-run-image-override": runImage(version, runImageTag, stack),
    }

def runImage(version, runImageTag, stack):
    # TODO(b/371521232): Newer runtimes do not publish to gcr.
    if version == "313":
        return "us-docker.pkg.dev/serverless-runtimes/google-22-full/runtimes/python313:latest"

    if stack != "":
        if runImageTag != "":
            return "us-docker.pkg.dev/gae-runtimes-private/" + stack + "/runtimes/python" + version + ":" + runImageTag
        else:
            return "gcr.io/gae-runtimes/buildpacks/python" + version + "/run"

    if runImageTag != "":
        return "us-docker.pkg.dev/gae-runtimes-private/gcp/buildpacks/python" + version + "/run:" + runImageTag
    else:
        return "gcr.io/gae-runtimes/buildpacks/python" + version + "/run"
