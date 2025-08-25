"""Utility macros for buildpacks."""

load("@rules_pkg//pkg:mappings.bzl", "pkg_mklink")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")
load("@bazel_skylib//rules:write_file.bzl", "write_file")

def buildpack(name, executables, prefix, version, api = "0.9", srcs = None, extension = "tgz", strip_prefix = ".", visibility = None):
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

def buildpack_using_runner(
        name,
        prefix,
        version,
        buildpack_id,
        api = "0.9",
        srcs = None,
        extension = "tgz",
        strip_prefix = ".",
        visibility = None):
    """Macro to create a single buildpack as a tgz or tar archive.

      The result is a tar or tgz archive with a buildpack descriptor
      (`buildpack.toml`) and interface scripts (bin/detect, bin/build).

      As this is a macro, the actual target name for the buildpack is `name_using_runner.extension`.
      The builder.toml spec allows either tar or tgz archives.

      Args:
        name: Base name for the targets.
        prefix: Namespace prefix.
        version: Buildpack version.
        buildpack_id: The full unique ID of the buildpack (e.g., google.nodejs.runtime).
        api: Buildpacks API version.
        srcs: Additional files to include.
        extension: Archive extension ("tgz" or "tar").
        strip_prefix: Prefix to strip from srcs.
        visibility: Target visibility.
      """
    descriptor_target_name = name + "_using_runner.descriptor"
    descriptor_output_filename = name + "_using_runner.buildpack.toml"

    _buildpack_descriptor(
        name = descriptor_target_name,
        api = api,
        version = version,
        prefix = prefix,
        bp_name = name,
        output = descriptor_output_filename,
    )

    if not srcs:
        srcs = []

    detect_script_name = name + "_detect_script"
    write_file(
        name = detect_script_name,
        out = "detect.sh",
        content = ["""#!/usr/bin/env bash
    /usr/local/bin/runner -buildpack="{id}" -phase="detect" "$@"
    """.format(id = buildpack_id)],
        is_executable = True,
    )

    build_script_name = name + "_build_script"
    write_file(
        name = build_script_name,
        out = "build.sh",
        content = ["""#!/usr/bin/env bash
    /usr/local/bin/runner -buildpack="{id}" -phase="build" "$@"
    """.format(id = buildpack_id)],
        is_executable = True,
    )

    pkg_mklink(
        name = name + "_detect_link",
        link_name = "bin/detect",
        target = "../detect.sh",
    )

    pkg_mklink(
        name = name + "_build_link",
        link_name = "bin/build",
        target = "../build.sh",
    )

    pkg_tar(
        name = name + "_using_runner",
        extension = extension,
        srcs = [
            descriptor_target_name,
            detect_script_name,
            build_script_name,
            name + "_detect_link",
            name + "_build_link",
        ] + srcs,
        files = {
            ":" + descriptor_output_filename: "buildpack.toml",
        },
        mode = "0755",
        package_dir = "/",
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

def builder(
        name,
        image,
        descriptor = "builder.toml",
        buildpacks = None,
        groups = None,
        visibility = None,
        builder_template = None,
        stack = None):
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
      builder_template: if builder.toml needs to be generated for input stack
      stack: Either of google.24.full or google.gae.22 or google.gae.18 representing ubuntu-24 or ubuntu-22 or ubuntu-18 stacks
    """
    srcs = []

    # Determine the builder descriptor source.
    # If a builder template and stack are provided, generate a custom descriptor.
    # Otherwise, use the default descriptor.
    if builder_template and stack:
        srcs.append(_generate_builder_descriptor(name, descriptor, builder_template, stack))
    else:
        srcs.append(descriptor)

    srcs += buildpacks if buildpacks else []

    deps = _package_buildpack_groups(name, groups) if groups else []

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

def _generate_builder_descriptor(name, descriptor, builder_template, stack):
    """Generates a builder descriptor from a template for a specific stack."""

    stack_to_gae_stack = {
        "google.gae.18": "google-gae-18",
        "google.gae.22": "google-gae-22",
        "google.24.full": "google-24-full",
    }
    gae_stack = stack_to_gae_stack.get(stack)
    image_prefix = "gcr.io/gae-runtimes/buildpacks/stacks/{}/".format(gae_stack)
    build_image = image_prefix + "build"
    run_image = image_prefix + "run"

    # Transform stack_id to google.24 for google.24.full.
    transformed_stack_id = stack
    if stack == "google.24.full":
        transformed_stack_id = "google.24"

    _builder_descriptor(
        name = name + ".descriptor",
        stack_id = transformed_stack_id,
        stack_build_image = build_image,
        stack_run_image = run_image,
        template = builder_template,
        output = name + "/" + descriptor,
    )
    return name + ".descriptor"

def _package_buildpack_groups(name, groups):
    """Packages buildpacks into groups."""

    deps = []
    for (k, v) in groups.items():
        pkg_tar(name = name + "_" + k, srcs = v, package_dir = k)
        deps.append(name + "_" + k)
    return deps

def _builder_descriptor_impl(ctx):
    ctx.actions.expand_template(
        output = ctx.outputs.output,
        substitutions = {
            "${STACK_ID}": ctx.attr.stack_id,
            "${STACK_BUILD_IMAGE}": ctx.attr.stack_build_image,
            "${STACK_RUN_IMAGE}": ctx.attr.stack_run_image,
        },
        template = ctx.file.template,
    )

_builder_descriptor = rule(
    implementation = _builder_descriptor_impl,
    attrs = {
        "stack_id": attr.string(mandatory = True),
        "stack_build_image": attr.string(mandatory = True),
        "stack_run_image": attr.string(mandatory = True),
        "template": attr.label(
            default = ":buildpack.toml.template",
            allow_single_file = True,
        ),
        "output": attr.output(mandatory = True),
    },
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
