"""
Binary dotnet/runtime buildpack detects .NET applications
and install the corresponding version of .NET runtime.
"""

import os
from typing import Dict, List, Optional

import pkg.dotnet as dotnet  # type: ignore
import pkg.env as env  # type: ignore
import pkg.gcpbuildpack as gcp  # type: ignore
import pkg.runtime as runtime  # type: ignore
from buildpacks.libcnb.v2 import Layer, Environment

sdk_layer_name = "sdk"
dev_mode_key = "devmode"

def detect_fn(ctx: gcp.Context) -> Optional[gcp.DetectResult]:
    """
    Detect function for .NET SDK buildpack.

    Args:
        ctx: The context object containing build information and utilities.

    Returns:
        A DetectResult indicating whether the buildpack should be used.
    """
    if runtime.check_override("dotnet"):
        return None

    try:
        files = dotnet.project_files(ctx, ".")
        if files:
            return gcp.OptIn(f"found project files: {', '.join(files)}")
    except Exception as e:
        logger.error(f"Error checking project files: {e}")
        raise

    return gcp.OptOut("no project files or .dll files found")

def build_fn(ctx: gcp.Context) -> None:
    """
    Build function for .NET SDK buildpack.

    Args:
        ctx: The context object containing build information and utilities.
    """
    try:
        sdk_version = dotnet.get_sdk_version(ctx)
        is_dev_mode = env.is_dev_mode()
        build_sdk_layer(ctx, sdk_version, is_dev_mode)
    except Exception as e:
        logger.error(f"Error building SDK layer: {e}")
        raise

def build_sdk_layer(
    ctx: gcp.Context,
    version: str,
    is_dev_mode: bool
) -> None:
    """
    Build the SDK layer for the .NET runtime.

    Args:
        ctx: The context object containing build information and utilities.
        version: The version of the .NET SDK to install.
        is_dev_mode: Whether the buildpack is running in development mode.
    """
    try:
        sdkl = ctx.layer(
            sdk_layer_name,
            gcp.BuildLayer,
            gcp.CacheLayer,
            gcp.LaunchLayerIfDevMode
        )
    except Exception as e:
        logger.error(f"Error creating {sdk_layer_name} layer: {e}")
        raise

    current_dev_mode = ctx.get_metadata(sdkl, dev_mode_key)
    if str(is_dev_mode) != current_dev_mode:
        try:
            ctx.clear_layer(sdkl)
        except Exception as e:
            logger.error(f"Error clearing layer {sdkl.name}: {e}")
            raise

    try:
        runtime.install_tarball_if_not_cached(
            ctx,
            runtime.dotnet_sdk,
            version,
            sdkl
        )
    except Exception as e:
        logger.error(f"Error installing SDK tarball: {e}")
        raise

    try:
        set_sdk_env_vars(ctx, sdkl, is_dev_mode)
    except Exception as e:
        logger.error(f"Error setting SDK environment variables: {e}")
        raise

    ctx.set_metadata(sdkl, dev_mode_key, str(is_dev_mode))

def set_sdk_env_vars(
    ctx: gcp.Context,
    sdkl: Layer,
    is_dev_mode: bool
) -> None:
    """
    Set the environment variables for the SDK layer.

    Args:
        ctx: The context object containing build information and utilities.
        sdkl: The SDK layer to modify.
        is_dev_mode: Whether the buildpack is running in development mode.
    """
    if dotnet.requires_globalization_invariant(ctx):
        sdkl.build_environment.default(
            "DOTNET_SYSTEM_GLOBALIZATION_INVARIANT",
            "1"
        )

    cap = ctx.capability(dotnet.skip_env_variables_assignment_capability)
    if cap:
        try:
            skip_vars = cap.skip_variables  # type: ignore
            skip_vars(ctx, sdkl)
        except Exception as e:
            logger.error(f"Error skipping environment variable assignment: {e}")
            raise

    if is_dev_mode:
        set_sdk_env_vars_dev_mode(sdkl)
    else:
        set_sdk_env_vars_for_build(sdkl)

def set_sdk_env_vars_dev_mode(sdkl: Layer) -> None:
    """
    Set environment variables specific to development mode.

    Args:
        sdkl: The SDK layer to modify.
    """
    sdkl.shared_environment.default("DOTNET_ROOT", sdkl.path)
    sdkl.shared_environment.prepend(
        "PATH",
        os.sep,
        sdkl.path
    )
    sdkl.launch_environment.default(
        "DOTNET_RUNNING_IN_CONTAINER",
        "true"
    )

def set_sdk_env_vars_for_build(sdkl: Layer) -> None:
    """
    Set environment variables for build time.

    Args:
        sdkl: The SDK layer to modify.
    """
    sdkl.build_environment.default("DOTNET_ROOT", sdkl.path)
    sdkl.build_environment.prepend(
        "PATH",
        os.sep,
        sdkl.path
    )
