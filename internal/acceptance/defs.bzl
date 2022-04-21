load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Macros for running acceptance tests."""

def acceptance_test_suite(name, srcs, testdata, builder, structure_test_config = ":config.yaml", deps = None, **kwargs):
    """Macro to define an acceptance test.

    Args:
      name: the name of the test
      srcs: the test source files
      testdata: a build target for a directory containing sample test applications
      builder: a build target for a builder.tar to test
      structure_test_config: a build target for the structured container test's config file
      deps: additional test dependencies beyond the acceptance package
      **kwargs: this argument captures all additional arguments and forwards them to the generated go_test rule
    """

    short_name = remove_suffix(name, "_test")

    if deps == None:
        deps = []
    deps.append("//internal/acceptance")

    # A builder target is a full google3 path, the name of the builder, and then :builder.tar, the following
    # extracts the name of the builder.
    builder_name = remove_suffix(builder, ":builder.tar")
    builder_name = builder_name[builder_name.rindex("/") + 1:]

    go_test(
        name = name,
        size = "enormous",
        srcs = srcs,
        args = [
            "-test-data=$(location " + testdata + ")",
            "-structure-test-config=$(location " + structure_test_config + ")",
            "-builder-source=$(location " + builder + ")",
            "-builder-prefix=" + builder_name + "-" + short_name + "-acceptance-test-",
        ],
        data = [
            structure_test_config,
            builder,
            testdata,
        ],
        tags = [
            "local",
        ],
        deps = deps,
        **kwargs
    )

# Once bazel supports python 3.9, this function can be replaced with `value.removesuffix(suffix)`:
def remove_suffix(value, suffix):
    if value.endswith(suffix):
        value = value[:-len(suffix)]
    return value
