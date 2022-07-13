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

_check_script = "//tools/checktools:main"

def meta_buildpack(name, descriptor = "buildpack.toml", buildpacks = [], visibility = None):
    """Macro to create a meta-buildpack as a ${name}.cnb file.

      A meta-buildpack is a buildpack that encapsulates other buildpacks and defines order groups
      for those buildpacks. The primary output of this rule is a target of ${name}.cnb which is a
      packaged buildpack file.

    Args:
      name: The base name of all the generated files and rules
      descriptor: The descriptor file for the meta-buildpack
      buildpacks: A list of other buildpacks to includes as dependencies
      visibility: the visibility
    """

    buildpack_archive_filename = _gen_meta_buildpack_archive(name, descriptor)
    package_toml_filename = _gen_package_toml(name, buildpack_archive_filename, buildpacks)
    _package_buildpack(name, buildpack_archive_filename, package_toml_filename, buildpacks, visibility)

# create the ${name}.cnb file
def _package_buildpack(name, buildpack_archive_filename, package_toml_filename, buildpacks, visibility):
    script_target = "//tools:buildpack_package"
    native.genrule(
        name = name,
        srcs = [buildpack_archive_filename, package_toml_filename] + buildpacks,
        outs = [name + ".cnb"],
        local = 1,
        tools = [
            _check_script,
            script_target,
        ],
        cmd = "$(execpath " + _check_script + ") && $(execpath " + script_target + ") $@ $(location " + package_toml_filename + ")",
        visibility = visibility,
    )

# generate the package.toml file with relative paths to each of the ${buildpacks}
def _gen_package_toml(name, buildpack_archive, dependency_buildpacks):
    script_target = "//tools:create_package_toml"
    package_toml_name = "package.toml"
    native.genrule(
        name = name + "_" + package_toml_name,
        srcs = [buildpack_archive] + dependency_buildpacks,
        outs = [package_toml_name],
        local = 1,
        tools = [
            script_target,
        ],
        cmd = "$(execpath " + script_target + ") $@ $(execpath " + buildpack_archive + ") $(SRCS)",
    )
    return package_toml_name

# generate an archive for the meta buildpack's buildpack.toml
def _gen_meta_buildpack_archive(name, descriptor):
    buildpack_archive_name = name + "_archive"
    pkg_tar(
        name = buildpack_archive_name,
        extension = "tgz",
        srcs = [descriptor],
        strip_prefix = ".",
    )
    return buildpack_archive_name + ".tgz"

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
