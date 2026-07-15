"""Rules to convert tooling_versions.bzl and tooling_versions_mock.bzl to json."""

load(":tooling_versions.bzl", "TOOLING_VERSIONS")
load(":tooling_versions_mock.bzl", "MOCK_TOOLING_VERSIONS")

def _tooling_versions_impl(ctx):
    ctx.actions.write(
        output = ctx.outputs.out,
        content = json.encode(TOOLING_VERSIONS),
    )

tooling_versions_to_json_rule = rule(
    implementation = _tooling_versions_impl,
    attrs = {"out": attr.output(mandatory = True)},
)

def tooling_versions_to_json(name, out):
    """Converts TOOLING_VERSIONS to JSON."""
    tooling_versions_to_json_rule(name = name, out = out)

def _mock_tooling_versions_impl(ctx):
    ctx.actions.write(
        output = ctx.outputs.out,
        content = json.encode(MOCK_TOOLING_VERSIONS),
    )

mock_tooling_versions_to_json_rule = rule(
    implementation = _mock_tooling_versions_impl,
    attrs = {"out": attr.output(mandatory = True)},
)

def mock_tooling_versions_to_json(name, out):
    """Converts MOCK_TOOLING_VERSIONS to JSON."""
    mock_tooling_versions_to_json_rule(name = name, out = out)
