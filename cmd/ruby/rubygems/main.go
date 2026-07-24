# Copyright 2025 Google LLC
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

"""Implements ruby/bundle buildpack. The bundle buildpack installs dependencies using bundle."""

import os
import sys
import subprocess
import glob
from typing import Any, Optional, Tuple

def detect_fn(context: dict[str, Any]) -> tuple[bool, str]:
    """Detects if Rubygems should be used based on presence of Gemfile or gems.rb."""
    gemfile_path = os.path.join(context["application_root"], "Gemfile")
    if os.path.exists(gemfile_path):
        return (True, "Gemfile found")
    
    gems_rb_path = os.path.join(context["application_root"], "gems.rb")
    if os.path.exists(gems_rb_path):
        return (True, "gems.rb found")
    
    return (False, "no Gemfile or gems.rb found")

def build_fn(context: dict[str, Any]) -> None:
    """Builds Rubygems layer and installs dependencies."""
    # Create layer directory structure
    layer_name = "rubygems"
    layer_path = os.path.join(context["layers_dir"], layer_name)
    os.makedirs(layer_path, exist_ok=True)

    if context.get("capabilities", {}).get(" GemsInstallerCapability") == False:
        print("GemsInstaller capability is disabled. Skipping installation.")
        return

    # Install Rubygems and Bundler
    install_rubygems(context, layer_path)
    
    # Handle Bundler 1.x compatibility for older Ruby versions
    if supports_bundler1(context):
        if is_using_bundler1(context):
            install_bundler1(context, layer_path)

    # Configure environment variables
    shared_env = context["shared_env"]
    shared_env.setdefault("RUBYLIB", os.path.join(layer_path, "lib"))
    shared_env.setdefault("GEM_PATH", f"{layer_path}:$GEM_PATH")
    shared_env["PATH"] = f"{os.path.join(layer_path, 'bin')}{os.pathsep}{shared_env.get('PATH', '')}"
    shared_env.setdefault("BUNDLE_DISABLE_EXEC_LOAD", "1")

def supports_bundler1(context: dict[str, Any]) -> bool:
    """Checks if current Ruby version supports Bundler 1.x."""
    return context["ruby_version"].startswith(("2.", "3."))

def is_using_bundler1(context: dict[str, Any]) -> bool:
    """Checks if the project uses Bundler 1.x based on lock file."""
    lock_file = ""
    gemfile_lock_path = os.path.join(context["application_root"], "Gemfile.lock")
    if os.path.exists(gemfile_lock_path):
        lock_file = gemfile_lock_path
    else:
        gems_locked_path = os.path.join(context["application_root"], "gems.locked")
        if os.path.exists(gems_locked_path):
            lock_file = gems_locked_path

    if not lock_file:
        return False

    with open(lock_file, "r", encoding="utf-8") as f:
        for line in f:
            if line.startswith("BUNDLED WITH"):
                version_line = next(f).strip()
                return version_line.startswith("1.")

def install_rubygems(context: dict[str, Any], layer_path: str) -> None:
    """Installs Rubygems and Bundler into the specified layer."""
    rubygems_version, bundler_version = get_versions(context)
    
    # Download and extract Rubygems
    temp_dir = os.path.join(layer_path, "tmp")
    os.makedirs(temp_dir, exist_ok=True)
    
    try:
        rubygems_url = f"https://rubygems.org/rubygems/rubygems-{rubygems_version}.tgz"
        subprocess.run(["wget", "-O", "-", rubygems_url], 
                      cwd=temp_dir,
                      check=True,
                      shell=False)
        
        subprocess.run(["tar", "xzf", os.path.join(temp_dir, f"rubygems-{rubygems_version}.tgz")],
                      cwd=temp_dir,
                      check=True,
                      shell=False)
        
        # Install Rubygems
        setup_script = os.path.join(temp_dir, f"rubygems-{rubygems_version}", "setup.rb")
        subprocess.run([sys.executable, setup_script, "-E", "--no-document", 
                        "--destdir", layer_path, "--prefix", "/"],
                      check=True,
                      shell=False)
        
    finally:
        # Cleanup temp files
        if os.path.exists(temp_dir):
            subprocess.run(["rm", "-rf", temp_dir], check=True, shell=False)

def install_bundler1(context: dict[str, Any], layer_path: str) -> None:
    """Installs Bundler 1.x compatibility layer."""
    print(f"Installing bundler {bundler1_version} for Rubygems compatibility")
    
    subprocess.run(["gem", "install", f"bundler:{bundler1_version}", "--no-document"],
                  env={"GEM_PATH": layer_path, "GEM_HOME": layer_path},
                  check=True,
                  shell=False)
    
    # Remove conflicting Bundler 2.x files
    bundler_dir = os.path.join(layer_path, "lib", "bundler")
    if os.path.exists(bundler_dir):
        subprocess.run(["rm", "-rf", bundler_dir], check=True, shell=False)
        
    bundler_rb = os.path.join(layer_path, "lib", "bundler.rb")
    if os.path.exists(bundler_rb):
        subprocess.run(["rm", "-f", bundler_rb], check=True, shell=False)

def get_versions(context: dict[str, Any]) -> tuple[str, str]:
    """Determines appropriate Rubygems and Bundler versions based on Ruby version."""
    ruby_version = context["ruby_version"]
    
    if ruby_version.startswith("2.5"):
        return ("3.2.26", "2.2.26")
        
    if ruby_version.startswith("4.0"):
        return ("4.0.3", "4.0.3")
        
    return ("3.3.15", "2.3.15")

def copy_default_gemspecs(layer_path: str) -> None:
    """Copies default gem specifications into the layer."""
    default_specs_dir = os.path.join(layer_path, "specifications", "default")
    specs_dir = os.path.join(layer_path, "specifications")
    
    for gemspec in glob.glob(os.path.join(default_specs_dir, "*.gemspec")):
        dest = os.path.join(specs_dir, os.path.basename(gemspec))
        print(f"Copying default gemspec {gemspec} to {dest}")
        subprocess.run(["cp", gemspec, dest], check=True, shell=False)
