load("@io_bazel_rules_go//go:def.bzl", "go_test")

"""Utility macros for buildpacks."""

load("@rules_pkg//pkg:mappings.bzl", "pkg_mklink")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

def buildpack(name, executables, prefix, version, api = "0.8", srcs = None, extension = "tgz", strip_prefix = ".", visibility = None):
    """Macro to create a single buildpack as a tgz or tar archive.

    The result is a tar or tgz archive with a buildpack descriptor
    (`buildpack.toml`) and interface scripts (bin/detect, bin/build).

    As this is a macro, the actual target name for the buildpack is `name.extension`.
    The builder.toml spec allows either tar or tgz archives.

    Args:
      name: the base name of the tar archive
      srcs: list of other files to include
      prefix: the language name or group used as a namespace in the buildpack ID
      version: the version of the buildpack
      api: the buildpacks API version
      executables: list of labels of buildpack binaries
      strip_prefix: by default preserves the paths of srcs
      extension: tgz by default
      visibility: the visibility
    """

    if len(executables) != 1:
        fail("You must provide exactly one buildpack executable")

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
    _buildpack_descriptor(
        name = name + ".descriptor",
        api = api,
        version = version,
        prefix = prefix,
        bp_name = name,
        output = "buildpack.toml",
    )

    if not srcs:
        srcs = []
    pkg_tar(
        name = name,
        extension = extension,
        srcs = [
            name + ".descriptor",
            "_link_build" + name,
            "_link_detect" + name,
        ] + srcs,
        files = {
            executables[0]: "/bin/main",
        },
        strip_prefix = strip_prefix,
        visibility = visibility,
    )

def _buildpack_descriptor_impl(ctx):
    ctx.actions.expand_template(
        output = ctx.outputs.output,
        substitutions = {
            "${API}": ctx.attr.api,
            "${VERSION}": ctx.attr.version,
            "${ID}": "google.{prefix}.{name}".format(
                prefix = ctx.attr.prefix,
                name = ctx.attr.bp_name.replace("_", "-"),
            ),
            "${NAME}": "{prefix} - {name}".format(
                prefix = _pretty_prefix(ctx.attr.prefix),
                name = ctx.attr.bp_name.replace("_", " ").title(),
            ),
        },
        template = ctx.file._template,
    )

_buildpack_descriptor = rule(
    implementation = _buildpack_descriptor_impl,
    attrs = {
        "api": attr.string(mandatory = True),
        "version": attr.string(mandatory = True),
        "bp_name": attr.string(mandatory = True),
        "prefix": attr.string(mandatory = True),
        "output": attr.output(mandatory = True),
        "_template": attr.label(
            default = ":buildpack.toml.template",
            allow_single_file = True,
        ),
    },
)

def _pretty_prefix(prefix):
    """Helper function to convert a buildpack prefix into a human readable name.

    Args:
      prefix: the namespace used in the buildpack id (eg dotnet, nodejs).
    """
    if prefix == "dotnet":
        return ".NET"
    if prefix == "php":
        return "PHP"
    if prefix == "nodejs":
        return "Node.js"
    if prefix == "cpp":
        return "C++"
    return prefix.title()

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

def buildpackage(name, buildpacks, descriptor = "buildpack.toml", visibility = None):
    """Macro to create a set of targets for a meta-buildpack containing the specified buildpacks.

    The result is a meta-builpack packages as a ".cnb" file created via the `pack buildpack package`
    command.

    `name` and `name.cnb`:
        Creates meta-buildpack persisted to disk as a ".cnb" file.

    `name.package`:
        Creates a package TOML file describing the contents of the meta-buildpack.

    `name.tar` and `name.tar.tar`:
        Creates source tarball for the meta-buildpacks that includes the specified buildpacks, the
        package TOML, and the buildpack TOML.

    Args:
      name: the name of the buildpackage to create.
      descriptor: path to the `buildpack.toml`
      buildpacks: list of labels to buildpacks (tar or tgz archives)
      visibility: the visibility
    """

    files = {
        descriptor: descriptor,
        "package.toml": "package.toml",
    }
    manifest = '''[buildpack]
  uri = "./"'''

    for b in buildpacks:
        # add the buildpack to the tarball fileset at the namespaced filepath
        files[b] = _buildpack_filepath(b)

        # add the buildpack to manifest at the namespaced filepath
        manifest += '''
[[dependencies]]
  uri = "./{tarname}"'''.format(tarname = _buildpack_filepath(b))

    native.genrule(
        name = name + ".package",
        outs = ["package.toml"],
        cmd = "echo '{manifest}' > $@".format(manifest = manifest),
    )

    pkg_tar(
        name = name + ".tar",
        extension = "tar",
        files = files,
        visibility = visibility,
    )

    # `name.image` and `name.sha` rules.
    native.genrule(
        name = name,
        srcs = [name + ".tar"],
        outs = [name + ".cnb"],
        local = 1,
        tools = [
            "//tools/checktools:main",
            "//tools:create_buildpackage",
        ],
        cmd = """$(execpath {check_script}) && $(execpath {create_script}) $(execpath {tar}) $@""".format(
            tar = name + ".tar",
            check_script = "//tools/checktools:main",
            create_script = "//tools:create_buildpackage",
        ),
    )

def _buildpack_filepath(symbol):
    """Helper function to convert a symbol pointing to a buildpack into a filepath.

    This prevents collisions by re-using the directory structure inside of the /cmd directory.

    Args:
      symbol: a build symbol of a buildpack to get a relative filepath for.
    """
    paths = symbol.split("/cmd/")
    if len(paths) != 2:
        fail("Buildpack symbol was not in cmd/: " + symbol)

    parts = paths[1].split(":")
    if len(parts) != 2:
        fail("Buildpack symbol was invalid: " + symbol)

    return parts[0] + ".tgz"
