"""Module for initializing arguments by nodejs version"""

load(":runtime.bzl", "flex_runtimes", "gae_runtimes", "gcf_runtimes", "version_to_stack")

# nodejs8 is decommissioned (was never available on flex)
flex_runtime_versions = {n: v for n, v in flex_runtimes.items()}
gae_runtime_versions = {n: v for n, v in gae_runtimes.items() if not v.startswith("8")}
gcf_runtime_versions = {n: v for n, v in gcf_runtimes.items() if not v.startswith("8")}
gcp_runtime_versions = dict(dict(flex_runtime_versions, **gae_runtime_versions), **gcf_runtime_versions)
nodejs_gcp_runtime_versions = [key for key in gcp_runtime_versions.keys()]

STACK_TO_BUILDER_PATH_MAP = {
    "google-18-full": "//builders/nodejs:builder.tar",
    "google-22-full": "//builders/nodejs:builder_22.tar",
    "google-24-full": "//builders/nodejs:builder_24.tar",
}

def get_runtime_to_builder_map():
    """Constructs and returns the RUNTIME_TO_BUILDER_MAP based on version_to_stack.

    Returns:
        dict: A dictionary mapping runtime versions (e.g., "nodejs18") to their
              corresponding builder file paths (e.g., "//path/to:builder.tar").
    """
    runtime_to_builder_map = {}
    for version, stack in version_to_stack.items():
        if stack in STACK_TO_BUILDER_PATH_MAP:
            runtime_to_builder_map[version] = STACK_TO_BUILDER_PATH_MAP[stack]
        else:
            fail("Error: No builder path defined in STACK_TO_BUILDER_PATH_MAP for stack: %s (for version %s)" % (stack, version))

    return runtime_to_builder_map

def nodejsargs(runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for nodejs tests

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for n, v in gae_runtimes.items():
        current_stack = ""
        if stack != "":
            current_stack = stack
        else:
            current_stack = version_to_stack.get(n)
        args[v] = newArgs(n, runImageTag, current_stack)

    return args

def newArgs(version, runImageTag, stack):
    return {
        "-run-image-override": runImage(version, runImageTag, stack),
    }

def runImage(version, runImageTag, stack):
    if runImageTag != "":
        return "us-docker.pkg.dev/gae-runtimes-private/" + stack + "/runtimes/" + version + ":" + runImageTag
    else:
        return "us-docker.pkg.dev/serverless-runtimes/" + stack + "/runtimes/" + version + ":latest"
