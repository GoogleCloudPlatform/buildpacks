import json
import os
import re
import shutil
from pathlib import Path
from typing import Optional, Tuple, List, Dict, Any
import xml.etree.ElementTree as ET

import cache  # Assuming cache is in the same directory or imported correctly
import devmode
import env
import gcpbuildpack as gcp
import runtime
import libcnb

# Constants
asp_dotnet_core = "Microsoft.AspNetCore.App"
env_sdk_version = "GOOGLE_DOTNET_SDK_VERSION"
google_min_22 = "google.min.22"
EnvRuntimeVersion = "GOOGLE_ASP_NET_CORE_VERSION"
PublishLayerName = "publish"
PublishOutputDirName = "bin"

SkipEnvVariablesAssignmentCapability = "dotnet.SkipEnvVariablesAssignmentCapability"


class MakerSkipEnvVariablesAssignment:
    def SkipVariables(self, ctx: gcp.Context, rtl: libcnb.Layer) -> None:
        pass


PublisherCapability = "dotnet.PublisherCapability"


class Publisher:
    def Publish(self, ctx: gcp.Context, proj: str, build_args: str) -> None:
        raise NotImplementedError()


class MakerDotnetPublisher:
    def Publish(self, ctx: gcp.Context, proj: str, build_args: str) -> None:
        Publish(ctx, proj, build_args, False)


cache_tag = "prod dependencies"
dependency_hash_key = "dependency_hash"
version_key = "version"


def Publish(
    ctx: gcp.Context,
    proj: str,
    build_args: str,
    use_layer: bool
) -> None:
    output_directory = ""
    pkg_layer: Optional[libcnb.Layer] = None
    bin_layer: Optional[libcnb.Layer] = None

    if use_layer:
        ctx.Log("Installing application dependencies.")
        pkg_layer, err = ctx.Layer("packages", gcp.BuildLayer, gcp.CacheLayer)
        if err is not None:
            raise RuntimeError(f"Creating layer failed: {err}")

        cached, err = CheckCache(ctx, pkg_layer)
        if err is not None:
            raise RuntimeError(f"Checking cache failed: {err}")
        if cached:
            ctx.CacheHit(cache_tag)
        else:
            ctx.CacheMiss(cache_tag)

        bin_layer, err = ctx.Layer(PublishLayerName, gcp.BuildLayer, gcp.LaunchLayer)
        if err is not None:
            raise RuntimeError(f"Creating layer failed: {err}")

        output_directory = str(Path(bin_layer.Path) / PublishOutputDirName)

        proj_path = Path(ctx.ApplicationRoot()) / proj
        if not proj_path.is_file():
            ctx.Warn("No project file found; skipping publish.")
            return

        deleted, err = DeleteFolder(ctx, Path(ctx.ApplicationRoot()) / PublishOutputDirName)
        if err is not None:
            raise RuntimeError(f"Deleting upload bin failed: {err}")
        if deleted:
            ctx.Warn(f"A project file was uploaded, causing `dotnet publish` to be called, but the output bin folder already existed in application source. Deleting {output_directory}.")

    else:
        output_directory = str(Path(ctx.ApplicationRoot()) / PublishOutputDirName)

        global_json = Path(ctx.ApplicationRoot()) / "global.json"
        if global_json.exists():
            ctx.Log("Temporarily renaming global.json to global.json.bak to roll forward SDK build.")
            global_json.rename(global_json.with_suffix(".json.bak"))
            ctx.AddCleanup(lambda: global_json.with_suffix(".json.bak").rename(global_json))

    # Restore
    restore_cmd = ["dotnet", "restore"]
    if use_layer:
        restore_cmd.extend(["--packages", pkg_layer.Path])
    restore_cmd.append(proj)

    _, err = ctx.Exec(restore_cmd, env={"DOTNET_CLI_TELEMETRY_OPTOUT": "true"}, user_attribution=True)
    if err is not None:
        raise RuntimeError(f"Restore command failed: {err}")

    # Publish
    publish_cmd = [
        "dotnet",
        "publish",
        "-nologo",
        "--verbosity", "minimal",
        "--configuration", "Release",
        "--output", output_directory,
        "--no-restore"
    ]
    if use_layer:
        publish_cmd.extend(["--packages", pkg_layer.Path])
    publish_cmd.append(proj)

    if build_args:
        publish_cmd = ["/bin/bash", "-c"] + [" ".join(publish_cmd + build_args.split())]

    _, err = ctx.Exec(publish_cmd, env={"DOTNET_CLI_TELEMETRY_OPTOUT": "true"}, user_attribution=True)
    if err is not None:
        raise RuntimeError(f"Publish command failed: {err}")

    # Runtime Version
    runtime_version, err = GetRuntimeVersion(ctx, output_directory)
    if err is not None:
        raise RuntimeError(f"Getting runtime version failed: {err}")

    if use_layer:
        bin_layer.BuildEnvironment.Default(EnvRuntimeVersion, runtime_version)
    else:
        os.environ[EnvRuntimeVersion] = runtime_version

    # Symlink
    if use_layer:
        ConfigureBinSymlink(ctx, output_directory)

    # Entrypoint
    entrypoint = os.getenv(env.Entrypoint)
    if entrypoint:
        entrypoint = f"exec {entrypoint}"
    else:
        ep, err = Entrypoint(ctx, output_directory, proj)
        if err is not None:
            raise RuntimeError(f"Getting entrypoint failed: {err}")
        entrypoint = ep
        if use_layer:
            bin_layer.BuildEnvironment.Default(env.Entrypoint, entrypoint)

    if use_layer:
        bin_layer.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")

        if not devmode.Enabled(ctx):
            ctx.AddWebProcess(["/bin/bash", "-c", entrypoint])
            return

        ctx.AddWebProcess(["dotnet", "watch", "--project", proj, "run"])
        return

    # MakerMode
    ctx.AddWebProcess(["/bin/bash", "-c", entrypoint])


def CheckCache(
    ctx: gcp.Context,
    layer: libcnb.Layer
) -> Tuple[bool, Optional[Exception]]:
    project_files, err = ProjectFiles(ctx, ".")
    if err is not None:
        return False, err

    global_json_path = Path(ctx.ApplicationRoot()) / "global.json"
    if global_json_path.exists():
        project_files.append(str(global_json_path))

    result, err = ctx.Exec(["dotnet", "--version"])
    if err is not None:
        return False, err
    current_version = result.Stdout

    hash_value, cached, err = cache.HashAndCheck(
        ctx,
        layer,
        dependency_hash_key,
        strings=[current_version],
        files=project_files
    )
    if err is not None:
        return False, err

    if cached:
        return True, None

    cache.Add(ctx, layer, dependency_hash_key, hash_value)
    ctx.SetMetadata(layer, version_key, current_version)
    return False, None


def DeleteFolder(
    ctx: gcp.Context,
    folder_path: Path
) -> Tuple[bool, Optional[Exception]]:
    if not folder_path.exists():
        return False, None

    try:
        shutil.rmtree(folder_path)
        return True, None
    except Exception as e:
        return False, e


def ConfigureBinSymlink(
    ctx: gcp.Context,
    bin_layer_path: str
) -> Optional[Exception]:
    link_target = Path(ctx.ApplicationRoot()) / PublishOutputDirName

    deleted, err = DeleteFolder(ctx, link_target)
    if err is not None:
        return f"Deleting {link_target}: {err}"
    if deleted:
        ctx.Warn(f"Deleted folder: {link_target}")
    else:
        ctx.Warn(f"Not deleting folder: {link_target}")

    try:
        os.symlink(bin_layer_path, link_target)
    except Exception as e:
        return f"Linking {bin_layer_path}: {e}"
    return None


def AssemblyName(ctx: gcp.Context, proj: str) -> Tuple[str, Optional[Exception]]:
    project_content, err = ReadProjectFile(ctx, proj)
    if err is not None:
        return "", RuntimeError(f"Reading project file failed: {err}")

    assembly_names = []
    for pg in project_content.PropertyGroups:
        if pg.AssemblyName:
            assembly_names.append(pg.AssemblyName)

    if len(assembly_names) != 1:
        return "", gcp.UserError(f"Expected exactly one AssemblyName, found {assembly_names}")
    
    return assembly_names[0], None


def Entrypoint(
    ctx: gcp.Context,
    bin_dir: str,
    proj: str
) -> Tuple[str, Optional[Exception]]:
    ctx.Log(f"Determining entrypoint from output directory {bin_dir} and project file {proj}")
    p = Path(proj).stem

    ep, err = EntrypointCmd(ctx, os.path.join(bin_dir, p))
    if err is not None:
        return "", RuntimeError(f"Getting entrypoint command failed: {err}")

    if ep:
        return ep, None

    an, err = AssemblyName(ctx, proj)
    if err is not None:
        return "", RuntimeError(f"Getting assembly name failed: {err}")

    ep, err = EntrypointCmd(ctx, os.path.join(bin_dir, an))
    if err is not None or not ep:
        return "", gcp.UserError("Unable to find executable produced from {proj}, try setting the AssemblyName property".format(proj=proj))
    
    return ep, None


def EntrypointCmd(
    ctx: gcp.Context,
    exe_path: str
) -> Tuple[str, Optional[Exception]]:
    dll_path = exe_path + ".dll"
    if not Path(dll_path).exists():
        return "", None

    dir_path = os.path.dirname(dll_path)
    try:
        rel_dir = os.path.relpath(dir_path, ctx.ApplicationRoot())
        if not rel_dir.startswith(".."):
            return f"exec dotnet {os.path.join(rel_dir, os.path.basename(dll_path))}", None
        else:
            return f"cd {dir_path} && exec dotnet {os.path.basename(dll_path)}", None
    except Exception as e:
        return "", RuntimeError(f"Constructing entrypoint command failed: {e}")


latest_dotnet_sdk_version_per_stack = {
    runtime.Ubuntu2204: "8.*.*",
    runtime.Ubuntu2404: "10.*.*",
}

proj_re = re.compile(r'(?i)\.(cs|fs|vb)proj$')


def ProjectFiles(
    ctx: gcp.Context,
    dir_path: str
) -> Tuple[List[str], Optional[Exception]]:
    project_files = []
    
    try:
        for root, dirs, files in os.walk(dir_path):
            for file in files:
                if proj_re.search(file):
                    project_files.append(os.path.join(root, file))
        return project_files, None
    except Exception as e:
        return [], RuntimeError(f"Finding project files failed: {e}")


class Project:
    def __init__(self):
        self.PropertyGroups = []
        self.ItemGroups = []

class PropertyGroup:
    def __init__(self):
        self.AssemblyName = ""
        self.TargetFramework = ""
        self.TargetFrameworks = ""

class ItemGroup:
    def __init__(self):
        self.PackageReferences = []

class PackageReference:
    def __init__(self):
        self.Include = ""
        self.Version = ""


def ReadProjectFile(
    ctx: gcp.Context,
    proj_path: str
) -> Tuple[Project, Optional[Exception]]:
    try:
        with open(proj_path, 'r') as f:
            content = f.read()
        return read_project_file(content, proj_path)
    except Exception as e:
        return Project(), RuntimeError(f"Reading project file failed: {e}")


def read_project_file(
    xml_content: str,
    proj_path: str
) -> Tuple[Project, Optional[Exception]]:
    try:
        root = ET.fromstring(xml_content)
        project = Project()
        
        for pg_elem in root.findall('PropertyGroup'):
            pg = PropertyGroup()
            pg.AssemblyName = pg_elem.findtext('AssemblyName', '')
            pg.TargetFramework = pg_elem.findtext('TargetFramework', '')
            pg.TargetFrameworks = pg_elem.findtext('TargetFrameworks', '')
            project.PropertyGroups.append(pg)
            
        for ig_elem in root.findall('ItemGroup'):
            ig = ItemGroup()
            for pr_elem in ig_elem.findall('PackageReference'):
                pr = PackageReference()
                pr.Include = pr_elem.get('Include', '')
                pr.Version = pr_elem.get('Version', '')
                ig.PackageReferences.append(pr)
            project.ItemGroups.append(ig)
            
        return project, None
    except Exception as e:
        return Project(), gcp.UserError(f"Unmarshalling {proj_path}: {e}")


def BuildableDir() -> str:
    buildable = os.getenv(env.Buildable)
    if not buildable:
        return "."
    
    if Path(buildable).suffix.lower() in ('.csproj', '.fsproj', '.vbproj'):
        return os.path.dirname(buildable)
    return buildable


def RuntimeConfigJSONFiles(path_dir: str) -> Tuple[List[str], Optional[Exception]]:
    try:
        pattern = os.path.join(path_dir, "*runtimeconfig.json")
        return glob.glob(pattern), None
    except Exception as e:
        return [], RuntimeError(f"Finding runtime config files failed: {e}")


class RuntimeConfigJSON:
    def __init__(self):
        self.RuntimeOptions = runtimeOptions()

class runtimeOptions:
    def __init__(self):
        self.TFM = ""
        self.Framework = framework()
        self.Frameworks = []
        self.ConfigProperties = configProperties()

class framework:
    def __init__(self):
        self.Name = ""
        self.Version = ""

class configProperties:
    def __init__(self):
        self.SystemGCServer = False


def ReadRuntimeConfigJSON(path_file: str) -> Tuple[Optional[RuntimeConfigJSON], Optional[Exception]]:
    try:
        with open(path_file, 'r') as f:
            data = json.load(f)
        runtime_cfg = RuntimeConfigJSON()
        runtime_cfg.RuntimeOptions.TFM = data.get('runtimeOptions', {}).get('tfm', '')
        
        if 'framework' in data['runtimeOptions']:
            fw = framework()
            fw.Name = data['runtimeOptions']['framework'].get('name', '')
            fw.Version = data['runtimeOptions']['framework'].get('version', '')
            runtime_cfg.RuntimeOptions.Framework = fw
        
        for fwm in data['runtimeOptions'].get('frameworks', []):
            fw = framework()
            fw.Name = fwm.get('name', '')
            fw.Version = fwm.get('version', '')
            runtime_cfg.RuntimeOptions.Frameworks.append(fw)
            
        cp = configProperties()
        cp.SystemGCServer = data['runtimeOptions']['configProperties'].get('System.GC.Server', False)
        runtime_cfg.RuntimeOptions.ConfigProperties = cp
        
        return runtime_cfg, None
    except Exception as e:
        return None, RuntimeError(f"Reading {path_file} failed: {e}")


def GetSDKVersion(ctx: gcp.Context) -> Tuple[str, Optional[Exception]]:
    version = os.getenv(env_sdk_version)
    if version:
        ctx.Log(f"Using .NET Core SDK version from {env_sdk_version}: {version}")
        return version, None

    version = os.getenv(env.RuntimeVersion)
    if version:
        ctx.Log(f"Using .NET Core SDK version from {env.RuntimeVersion}: {version}")
        return version, None

    global_json = GetGlobalJSON(ctx.ApplicationRoot())
    if global_json and global_json.Sdk.Version:
        ctx.Log(f"Using .NET Core SDK version from global.json: {global_json.Sdk.Version}")
        return global_json.Sdk.Version, None

    os_name = runtime.OSForStack(ctx)
    version = latest_dotnet_sdk_version_per_stack.get(os_name)
    if not version:
        return "", gcp.UserError(f"Invalid stack for .NET runtime: {os_name}. Please use a supported stack")
    
    ctx.Log(f".NET SDK version not specified, using the latest available .NET SDK for the stack {os_name}")
    return version, None


def GetGlobalJSON(application_root: str) -> Optional[globalJSON]:
    global_json_path = os.path.join(application_root, "global.json")
    if not os.path.exists(global_json_path):
        return None
    
    try:
        with open(global_json_path, 'r') as f:
            data = json.load(f)
        sdk_version = data.get('sdk', {}).get('version', '')
        return globalJSON(Sdk=SDKVersion(version=sdk_version)) if sdk_version else None
    except Exception as e:
        return None


class SDKVersion:
    def __init__(self, version: str):
        self.Version = version

class globalJSON:
    def __init__(self, Sdk: SDKVersion):
        self.Sdk = Sdk


def FindProjectFile(ctx: gcp.Context) -> Tuple[str, Optional[Exception]]:
    proj = os.getenv(env.Buildable)
    if not proj:
        proj = "."
    
    try:
        if os.path.isdir(proj):
            project_files, err = ProjectFiles(ctx, proj)
            if err is not None:
                return "", RuntimeError(f"Finding project files failed: {err}")
            
            if len(project_files) != 1:
                return "", gcp.UserError(f"Expected exactly one project file in directory {proj}, found {project_files}")
                
            proj = project_files[0]
        return proj, None
    except Exception as e:
        return "", RuntimeError(f"Finding project file failed: {e}")


def GetRuntimeVersion(
    ctx: gcp.Context,
    dir_path: str
) -> Tuple[str, Optional[Exception]]:
    env_version = os.getenv(EnvRuntimeVersion)
    if env_version:
        ctx.Log(f"Determined runtime version from {EnvRuntimeVersion}: {env_version}")
        return env_version, None

    rt_cfg_version, rt_cfg_file, err = GetRuntimeVersionFromRtCfgDir(ctx, dir_path)
    if err is not None:
        return "", RuntimeError(f"{EnvRuntimeVersion} was not set; getting version from runtimeconfig.json failed: {err}")
    
    ctx.Log(f"Determined runtime version from {rt_cfg_file}: {rt_cfg_version}")
    return rt_cfg_version, None


def GetRuntimeVersionFromRtCfgDir(
    ctx: gcp.Context,
    dir_path: str
) -> Tuple[str, str, Optional[Exception]]:
    rt_cfg_files, err = RuntimeConfigJSONFiles(dir_path)
    if err is not None:
        return "", "", RuntimeError(f"Finding runtimeconfig.json failed: {err}")
    
    if len(rt_cfg_files) > 1:
        return "", "", RuntimeError(f"More than one runtimeconfig.json file found: {rt_cfg_files}")
    
    if not rt_cfg_files:
        return "", "", RuntimeError("No runtimeconfig.json file was found")
    
    ctx.Log(f"Found runtimeconfig file {rt_cfg_files[0]}")

    runtime_cfg, err = ReadRuntimeConfigJSON(rt_cfg_files[0])
    if err is not None:
        return "", rt_cfg_files[0], RuntimeError(f"Reading runtimeconfig.json failed: {err}")

    version = ""
    if runtime_cfg.RuntimeOptions.Framework.Name == asp_dotnet_core:
        version = runtime_cfg.RuntimeOptions.Framework.Version
    else:
        for fw in runtime_cfg.RuntimeOptions.Frameworks:
            if fw.Name == asp_dotnet_core:
                version = fw.Version
                break
    
    if not version:
        return "", rt_cfg_files[0], RuntimeError(f"Couldn't find runtime version for framework {asp_dotnet_core} from runtimeconfig.json")
    
    return version, rt_cfg_files[0], None


def RequiresGlobalizationInvariant(ctx: gcp.Context) -> bool:
    return ctx.StackID() == google_min_22
