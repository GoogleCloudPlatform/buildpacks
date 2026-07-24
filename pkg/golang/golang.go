# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Package golang contains Go buildpack library code.
"""

import os
import re
import sys
from typing import Optional, Tuple

import pkg.cache
import pkg.env
import pkg.fetch
import pkg.runtime
import pkg.version

GO_VERSION_REGEX = re.compile(r"^go version go(\d+(\.\d+){1,2})([a-z]+\d+)? .*$")
GO_MOD_VERSION_REGEX = re.compile(r"(?m)^\s*go\s+(\d+(\.\d+){1,2})\s*$")

const (
    # OutBin is the name of the final compiled binary produced by Go buildpacks.
    OutBin = "main"
    # BuildDirEnv is an environment variable that buildpacks can use to communicate the working directory to `go build`.
    BuildDirEnv = "GOOGLE_INTERNAL_BUILD_DIR"
    goPathLayerName = "gopath"
    goModCacheKey = "go-mod-sha"
    envGoVersion = "GOOGLE_GO_VERSION"
)

class GoBuilder:
    """
    Interface for building Go applications.
    """
    def build(self, ctx: 'Context', out_bin: str, buildable: str, workdir: str, gocache: str, flags: list[str]) -> None:
        pass

class MakerGolangBuilder(GoBuilder):
    """
    Implementation of GoBuilder interface for the maker tool.
    """
    def build(self, ctx: 'Context', out_bin: str, buildable: str, workdir: str, gocache: str, flags: list[str]) -> None:
        cmd = ["go", "build"] + flags + ["-o", out_bin, buildable]
        envs = ["GOOS=linux", "GOARCH=amd64"]
        ctx.exec_cmd(cmd, envs=envs, workdir=workdir)

def PerformBuild(ctx: 'Context', bin_layer: 'Layer', buildable: str, workdir: str, gocache: str, bld_flags: list[str]) -> Tuple[str, list[str]]:
    out_bin = os.path.join(bin_layer.path, OutBin)
    bld = ["go", "build"] + bld_flags + ["-o", out_bin, buildable]
    return execute_build(ctx, bin_layer, out_bin, buildable, workdir, gocache, bld_flags, bld)

def execute_build(ctx: 'Context', bin_layer: 'Layer', out_bin: str, buildable: str, workdir: str, gocache: str, bld_flags: list[str], bld: list[str]) -> Tuple[str, list[str]]:
    cap = ctx.get_capability(GoBuilderCapability)
    if cap:
        out_bin = "./" + OutBin
        bld = ["go", "build"] + bld_flags + ["-o", out_bin, buildable]
        if not isinstance(cap, GoBuilder):
            raise ValueError(f"capability {GoBuilderCapability} must implement GoBuilder")
        cap.build(ctx, out_bin, buildable, workdir, gocache, bld_flags)
    else:
        bin_layer.launch_env.prepend("PATH", os.pathsep, bin_layer.path)
        ctx.exec_cmd(bld, env={"GOCACHE": gocache}, workdir=workdir)
    return out_bin, bld

def print_tips_and_keep_stderr_tail(ctx: 'Context') -> callable:
    def message_producer(result: dict) -> str:
        if result["exit_code"] != 0 and (noGoFileError in result["stderr"] or cannotFindModuleError in result["stderr"]):
            ctx.tip(f"Tip: {env.Buildable} env var configures which Go package is built. Default is '.'")
        return keep_stderr_tail(result)
    return message_producer

def SupportsAppEngineApis(ctx: 'Context') -> bool:
    if IsGo111Runtime():
        return True
    return pkg.appengine.ApiEnabled(ctx)

def SupportsAutoVendor(ctx: 'Context') -> bool:
    return VersionMatches(ctx, ">=1.14.0")

def SupportsGoProxyFallback(ctx: 'Context') -> bool:
    return VersionMatches(ctx, ">=1.15.0")

def SupportsGoCleanModCache(ctx: 'Context') -> bool:
    return VersionMatches(ctx, ">=1.13.0")

def SupportsGoGet(ctx: 'Context') -> bool:
    v = RuntimeVersion(ctx)
    if not v or v == "":
        return False
    return VersionMatches(ctx, "<1.22.0", v)

def SupportsVendorModificaton(ctx: 'Context') -> bool:
    v = RuntimeVersion(ctx)
    if not v or v == "":
        return False
    return VersionMatches(ctx, "<1.23.0", v)

def VersionMatches(ctx: 'Context', version_range: str, go_versions: list[str] = []) -> bool:
    if len(go_versions) == 0:
        gomod_version = GoModVersion(ctx)
        if not gomod_version:
            return False
        v = gomod_version
    else:
        v = go_versions[0]
    
    if is_supported_unstable_go_version(v):
        if "rc" in v and "-rc" not in v:
            v = v.replace("rc", "-rc", 1)
    
    try:
        version = pkg.version.Version.parse(v)
    except ValueError as e:
        raise ValueError(f"unable to parse go.mod version string {v}: {e}")
    
    try:
        constraint = pkg.version.Constraint.parse(version_range)
    except ValueError as e:
        raise ValueError(f"unable to parse version range {version_range}: {e}")
    
    if not constraint.check(version):
        return False
    
    runtime_version = GoVersion(ctx)
    try:
        runtime_ver = pkg.version.Version.parse(runtime_version)
    except ValueError as e:
        raise ValueError(f"unable to parse Go version string {runtime_version}: {e}")
    
    return constraint.check(runtime_ver)

def GoVersion(ctx: 'Context') -> str:
    v = read_go_version(ctx)
    match = GO_VERSION_REGEX.match(v)
    if not match or len(match.groups()) < 2 or match.group(1) == "":
        raise ValueError(f"unable to find go version in {v}")
    return match.group(1)

def GoModVersion(ctx: 'Context') -> Optional[str]:
    gomod_path = os.path.join(ctx.application_root(), "go.mod")
    if not ctx.file_exists(gomod_path):
        return None
    content = ctx.read_file(gomod_path)
    match = GO_MOD_VERSION_REGEX.search(content)
    if not match or len(match.groups()) < 2 or match.group(1) == "":
        return None
    return match.group(1)

def read_go_version(ctx: 'Context') -> str:
    result = ctx.exec_cmd(["go", "version"])
    return result["stdout"]

def clean_mod_cache(ctx: 'Context') -> None:
    ctx.exec_cmd(["go", "clean", "-modcache"])

def NewGoWorkspaceLayer(ctx: 'Context') -> 'Layer':
    layer = ctx.create_layer(goPathLayerName, cache=True, launch_if_dev=True)
    layer.build_env.override("GOPATH", layer.path)
    layer.build_env.override("GO111MODULE", "on")
    layer.build_env.override("GOPROXY", "off")

    if not SupportsGoCleanModCache(ctx):
        layer.cache = False
        return layer

    hash_value, cached = pkg.cache.hash_and_check(ctx, layer, goModCacheKey, files=[go_mod_path(ctx)])
    if not cached:
        clean_mod_cache(ctx)
        pkg.cache.add(ctx, layer, goModCacheKey, hash_value)

    return layer

def go_mod_path(ctx: 'Context') -> str:
    return os.path.join(ctx.application_root(), "go.mod")

def ExecWithGoproxyFallback(ctx: 'Context', cmd: list[str], opts: list[dict]) -> dict:
    if "GOPROXY" in os.environ:
        return ctx.exec_cmd(cmd, opts)
    
    if SupportsGoProxyFallback(ctx):
        opts.append({"env": {"GOPROXY": "https://proxy.golang.org|direct"}})
        return ctx.exec_cmd(cmd, opts)
    
    result = ctx.exec_cmd(cmd, opts)
    if result["exit_code"] != 0:
        ctx.warn(f"Command {cmd} failed. Retrying with GOSUMDB=off GOPROXY=direct. Error: {result['error']}")
        opts.append({"env": {"GOSUMDB": "off", "GOPROXY": "direct"}})
        result = ctx.exec_cmd(cmd, opts)
    return result

def IsGo111Runtime() -> bool:
    return os.getenv(pkg.env.Runtime) == "go111"

def RuntimeVersion(ctx: 'Context') -> str:
    version = ""
    if env_go_version := os.getenv(envGoVersion):
        version = env_go_version
        ctx.log(f"Using runtime version from {envGoVersion}: {version}")
    elif runtime_version := os.getenv(pkg.env.RuntimeVersion):
        version = runtime_version
        ctx.log(f"Using runtime version from {pkg.env.RuntimeVersion}: {version}")
    else:
        os_name = pkg.runtime.os_for_stack(ctx)
        version, ok = latest_go_versions.get(os_name, ("", False))
        if not ok:
            raise ValueError(f"invalid stack for Go runtime: {os_name}")
        ctx.log(f"Go version not specified, using latest available Go runtime for the stack {os_name}")

    if ctx.get_capability(GoBuilderCapability):
        local_version = GoVersion(ctx)
        return local_version

    resolved_version = ResolveGoVersion(version)
    return resolved_version

def ResolveGoVersion(ver_constraint: str) -> str:
    if is_supported_unstable_go_version(ver_constraint) or is_exact_go_semver(ver_constraint):
        return ver_constraint
    
    releases = fetch_json(go_versions_url)
    versions = [r["version"].lstrip("go") for r in releases if r["stable"]]
    
    try:
        resolved = pkg.version.resolve_version(ver_constraint, versions)
    except ValueError as e:
        raise ValueError(f"invalid Go version specified: {ver_constraint}. You can refer to {go_versions_url} for a list of stable Go releases. Error: {e}")
    
    return resolved

def is_supported_unstable_go_version(constraint: str) -> bool:
    if constraint.count(".") == 1 and "rc" in constraint:
        return True
    return False

def is_exact_go_semver(constraint: str) -> bool:
    if constraint.count(".") not in (1, 2):
        return False
    try:
        pkg.version.Version.parse(constraint)
        return True
    except ValueError:
        return False
