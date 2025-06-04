"""Module for initializing arguments for acceptance tests"""

def get_runtime_to_builder_map(version_to_stack, language):
    """Constructs and returns the RUNTIME_TO_BUILDER_MAP based on version_to_stack.

    Args:
      version_to_stack: A dictionary mapping runtime versions to their stacks.
        e.g., {"go120": "google-22-full"}.
      language: The language for which to construct the builder path (e.g., "go", "nodejs").

    Returns:
        dict: A dictionary mapping runtime versions (e.g., "nodejs18") to their
              corresponding builder file paths (e.g., "//path/to:builder.tar").
    """
    stack_to_builder_path_map = {
        "google-18-full": "//builders/" + language + ":builder.tar",
        "google-22-full": "//builders/" + language + ":builder_22.tar",
        "google-24-full": "//builders/" + language + ":builder_24.tar",
    }
    runtime_to_builder_map = {}
    for version, stack in version_to_stack.items():
        if stack in stack_to_builder_path_map:
            runtime_to_builder_map[version] = stack_to_builder_path_map[stack]
        else:
            fail("Error: No builder path defined in stack_to_builder_path_map for stack: %s (for version %s)" % (stack, version))

    return runtime_to_builder_map

def languageargs(runtimes, version_to_stack, runImageTag = "", stack = ""):
    """Create a new key-value map of arguments for language tests

    Args:
      runtimes: A dictionary mapping runtime to their latest versions.
        e.g., {"nodejs22": "22.16.0"}.
      version_to_stack: A dictionary mapping runtime versions to their stacks.
        e.g., {"nodejs22": "google-22-full"}.
      runImageTag: The tag to use for the run image.
      stack: The stack to use for the run image.

    Returns:
        A key-value map of the arguments
    """
    args = {}
    for n, v in runtimes.items():
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
