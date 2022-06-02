load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Utility macros for buildpacks."""

load("@rules_pkg//pkg:mappings.bzl", "pkg_mklink")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

def buildpack(name, executables, descriptor = "buildpack.toml", srcs = None, extension = "tgz", strip_prefix = ".", visibility = None):
    """Macro to create a single buildpack as a tgz or tar archive.

    The result is a tar or tgz archive with a buildpack descriptor
    (`buildpack.toml`) and interface scripts (bin/detect, bin/build).

    As this is a macro, the actual target name for the buildpack is `name.extension`.
    The builder.toml spec allows either tar or tgz archives.

    Args:
      name: the base name of the tar archive
      descriptor: path to the `buildpack.toml`
      srcs: list of other files to include
      executables: list of labels of buildpack binaries
      strip_prefix: by default preserves the paths of srcs
      extension: tgz by default
      visibility: the visibility
    """

    pkg_mklink(
        name = "_link_build" + name,
        target = "main",
        link_name = "bin/build",
    )
    pkg_mklink(
        name = "_link_detect" + name,
        target = "main",
        link_name = "bin/detect",
    )

    # relocate binary into bin/, create symlinks
    pkg_tar(
        name = name + "_executables",
        package_dir = "bin",
        srcs = executables + [
            ":_link_build" + name,
            ":_link_detect" + name,
        ],
    )
    if not srcs:
        srcs = []
    pkg_tar(
        name = name,
        extension = extension,
        srcs = [descriptor] + srcs,
        deps = [name + "_executables"],
        strip_prefix = strip_prefix,
        visibility = visibility,
    )

def builder(name, image, descriptor = "builder.toml", buildpacks = None, groups = None, visibility = None):
    """Macro to create a set of targets for a builder with specified buildpacks.

    `name` and `name.tar`:
        Creates tar archive with a builder descriptor (`builder.toml`) and its
        associated buildpacks.  The buildpacks should either have unique names
        or be assigned to different groups.  The grouped buildpacks are placed
        in directories named by the key.  Both `buildpacks` and `groups` may
        be used simultaneously.

    `name.image` and `name.sha`:
        Creates a builder image based on the source from `name.tar` using pack
        and outputs the image SHA into the `name.sha` file. The builder will
        be named `image`.

    Args:
      name: the base name of the tar archive
      image: the name of the builder image
      descriptor: path to the `builder.toml`
      buildpacks: list of labels to buildpacks (tar or tgz archives)
      groups: dict(name -> list of labels to buildpacks);
        the buildpacks are grouped under a single-level directory named <key>
      visibility: the visibility
    """
    srcs = [descriptor]
    if buildpacks:
        srcs += buildpacks
    deps = []
    if groups:
        for (k, v) in groups.items():
            pkg_tar(name = name + "_" + k, srcs = v, package_dir = k)
            deps.append(name + "_" + k)

    # `name` and `name.tar` rules.
    pkg_tar(
        name = name,
        extension = "tar",
        srcs = srcs,
        deps = deps,
        visibility = visibility,
    )

    # `name.image` and `name.sha` rules.
    native.genrule(
        name = name + ".image",
        srcs = [name + ".tar"],
        outs = [name + ".sha"],
        local = 1,
        tools = [
            "//tools/checktools:main",
            "//tools:create_builder",
        ],
        cmd = """$(execpath {check_script}) && $(execpath {create_script}) {image} $(execpath {tar}) "{descriptor}" $@""".format(
            image = image,
            tar = name + ".tar",
            descriptor = descriptor,
            check_script = "//tools/checktools:main",
            create_script = "//tools:create_builder",
        ),
    )
