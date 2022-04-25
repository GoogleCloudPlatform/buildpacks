load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Macros for running acceptance tests."""

def acceptance_test_suite(name, srcs, testdata, builder, structure_test_config = ":config.yaml", args = None, deps = None, **kwargs):
    """Macro to define an acceptance test.

    Args:
      name: the name of the test
      srcs: the test source files
      testdata: a build target for a directory containing sample test applications
      builder: a build target for a builder.tar to test
      structure_test_config: a build target for the structured container test's config file
      args: additional arguments to be passed to the test binary beyond ones corresponding to the arguments to this function
      deps: additional test dependencies beyond the acceptance package
      **kwargs: this argument captures all additional arguments and forwards them to the generated go_test rule
    """

    go_test(
        name = name,
        size = "enormous",
        srcs = srcs,
        args = _build_args(args, name, testdata, builder, structure_test_config),
        data = [
            structure_test_config,
            builder,
            testdata,
        ],
        tags = [
            "local",
        ],
        gc_linkopts = [
            "-I",
            "/lib64/ld-linux-x86-64.so.2",
            "-extldflags=\"-Wl,--dynamic-linker,/lib64/ld-linux-x86-64.so.2\"",
        ],
        deps = _build_deps(deps),
        **kwargs
    )

def _build_args(args, name, testdata, builder, structure_test_config):
    short_name = _remove_suffix(name, "_test")
    builder_name = _extract_builder_name(builder)

    if args == None:
        args = []
    args.append("-test-data=$(location " + testdata + ")")
    args.append("-structure-test-config=$(location " + structure_test_config + ")")
    args.append("-builder-source=$(location " + builder + ")")
    args.append("-builder-prefix=" + builder_name + "-" + short_name + "-acceptance-test-")
    return args

def _extract_builder_name(builder):
    # A builder target is a full google3 path, the name of the builder, and then :builder.tar, the following
    # extracts the name of the builder.
    builder_name = _remove_suffix(builder, ":builder.tar")
    builder_name = builder_name[builder_name.rindex("/") + 1:]
    return builder_name

def _build_deps(deps):
    if deps == None:
        deps = []
    deps.append("//internal/acceptance")
    return deps

# Once bazel supports python 3.9, this function can be replaced with `value.removesuffix(suffix)`:
def _remove_suffix(value, suffix):
    if value.endswith(suffix):
        value = value[:-len(suffix)]
    return value
