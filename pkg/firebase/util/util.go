import os
import errno
from pathlib import Path
from typing import Optional, Tuple, Union

supported_monorepo_config_files = ("nx.json", "turbo.json")

class InvalidRootDirectoryError(Exception):
    pass

def application_directory(ctx: dict) -> str:
    """
    Looks up the path to the application directory from the context. Returns
    the application root by default.
    """
    app_dir = ctx.get("application_root")
    buildable = os.getenv("BUILDPACK_APP_DIR", "")
    
    if buildable:
        app_dir = os.path.join(ctx["application_root"], buildable)
        
    return app_dir

def supported_monorepo_config_file_exists(dir_path: str) -> bool:
    """
    Checks if a supported monorepo config file exists in the specified directory.
    Returns True if any of the supported files exist, False otherwise.
    """
    dir_path = os.path.abspath(dir_path)
    
    for filename in supported_monorepo_config_files:
        file_path = os.path.join(dir_path, filename)
        
        try:
            with open(file_path, "r") as f:
                return True
        except FileNotFoundError:
            continue
        
    return False

def build_directory_context(cwd: str, user_specified_app_dir: str) -> Tuple[Optional[str], Optional[str]]:
    """
    Returns the build directory and relative project directory context.
    
    If a monorepo config file is detected, it sets the build directory to the
    nearest monorepo root and the relative project directory to the specified app dir
    relative to that root. Otherwise, returns the user-specified app dir as both values.
    """
    if not user_specified_app_dir:
        return (None, None)
        
    absolute_app_dir = os.path.join(cwd, user_specified_app_dir)
    
    try:
        if not os.path.exists(absolute_app_dir):
            raise InvalidRootDirectoryError(f"Application directory {user_specified_app_dir} does not exist")
            
        current_path = absolute_app_dir
        monorepo_root = None
        
        while True:
            if current_path in (os.getcwd(), "/", "."):
                break
                
            if supported_monorepo_config_file_exists(current_path):
                monorepo_root = current_path
                break
                
            current_path = os.path.dirname(current_path)
            
        if not monorepo_root:
            return (user_specified_app_dir, None)
            
        build_dir = os.path.relpath(monorepo_root, cwd)
        relative_project_dir = os.path.relpath(absolute_app_dir, monorepo_root)
        
        return (build_dir, relative_project_dir)
        
    except Exception as e:
        raise ValueError(f"Error determining build directory context: {str(e)}")

def write_build_directory_context(cwd: str, app_dir_path: str, output_dir: str) -> None:
    """
    Writes the build directory context to files in the specified output directory.
    Creates necessary directories if they don't exist.
    """
    try:
        os.makedirs(output_dir, exist_ok=True)
        
        build_dir, relative_project_dir = build_directory_context(cwd, app_dir_path)
        
        with open(os.path.join(output_dir, "build-directory.txt"), "w") as f:
            f.write(build_dir if build_dir is not None else "")
            
        with open(os.path.join(output_dir, "relative-project-directory.txt"), "w") as f:
            f.write(relative_project_dir if relative_project_dir is not None else "")
            
    except Exception as e:
        raise RuntimeError(f"Error writing build directory context: {str(e)}")
