import os
import time
from typing import List, Optional
import logging
import fnmatch

from buildpack.gcpbuildpack import (
    DetectResult,
    OptOut,
    OptOutEnvNotSet,
    UserErrorf,
)
from buildpack.env import ClearSource as env_ClearSource
from buildpack.devmode import Enabled as devmode_Enabled
from buildpack.appstart import ConfigDir

default_exclusions = [ConfigDir]

def DetectFn(ctx: dict) -> tuple[Optional[DetectResult], Optional[str]]:
    """
    Determines if clear source buildpacks should opt out.
    Returns a detection result or nil if it shouldn't opt out.
    """
    if devmode_Enabled(ctx):
        return OptOut("development mode enabled"), None

    clear_source = os.getenv(env_ClearSource)
    if clear_source is not None:
        try:
            clear = bool(clear_source)
            if clear:
                return None, None
        except ValueError as e:
            return None, str(UserErrorf(f"parsing {env_ClearSource!r}: {e}"))
    
    return OptOutEnvNotSet(env_ClearSource), None

def BuildFn(ctx: dict, exclusions: List[str]) -> Optional[str]:
    """
    Clears the workspace while leaving exclusion patterns untouched.
    Exclusions are relative to the application directory.
    Returns an error if any occurs during clearing.
    """
    logging.info("Clearing source")
    
    start_time = time.time()
    try:
        # Perform the clear operation
        exclusions += default_exclusions
        paths, err = paths_to_remove(ctx, ctx["ApplicationRoot"](), exclusions)
        if err is not None:
            return f"Filtering paths: {err}"
        
        for path in paths:
            if os.path.exists(path):
                if os.path.isdir(path):
                    os.rmdir(path)
                else:
                    os.remove(path)
    finally:
        # Log the duration of the clear operation
        elapsed = time.time() - start_time
        logging.info(f"Clear source completed in {elapsed:.2f}s")
    
    return None

def paths_to_remove(ctx: dict, directory: str, exclusions: List[str]) -> tuple[List[str], Optional[str]]:
    """
    Returns a list of paths to remove, excluding those that match any pattern.
    Exclusions are relative to the given directory.
    Returns an error if any occurs during globbing.
    """
    try:
        # Get all items in the directory
        with os.scandir(directory) as entries:
            paths = [os.path.join(directory, entry.name) for entry in entries]
    except OSError as e:
        return [], str(e)
    
    filtered_paths = []
    for path in paths:
        remove = True
        for exclusion in exclusions:
            pattern = os.path.join(directory, exclusion)
            # Check if the current path matches any exclusion pattern
            if fnmatch.fnmatch(path, pattern):
                remove = False
                break
        
        if remove:
            filtered_paths.append(path)
    
    return filtered_paths, None
