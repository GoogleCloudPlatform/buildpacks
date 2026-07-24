# Copyright 2026 Google LLC
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

import json
import os
import sys
from pathlib import Path

def detect():
    # Restrict this feature behind ALPHA release track.
    if not os.environ.get("GOOGLE_CLOUD_RELEASE_TRACK", "").lower() == "alpha":
        print("Static runtimes feature is supported only on ALPHA release track.")
        return
    
    firebase_config_path = os.path.join(os.getcwd(), "firebase.json")
    try:
        with open(firebase_config_path, 'r') as f:
            config = json.load(f)
            public_dir = config.get('public', '')
            if public_dir:
                full_path = os.path.join(os.getcwd(), public_dir)
                if os.path.isdir(full_path):
                    print(f"Opted in via firebase.json (public: {public_dir})")
                    return
    except Exception as e:
        pass
    
    print("No valid firebase.json public asset folder found.")
    return

def build():
    # Create a layer for nginx configuration
    layer_name = "nginx_config"
    try:
        os.makedirs(layer_name, exist_ok=True)
    except Exception as e:
        print(f"Error creating layer {layer_name}: {e}")
        sys.exit(1)

    root_path = os.getcwd()
    fb_config = None

    firebase_config_path = os.path.join(os.getcwd(), "firebase.json")
    if os.path.isfile(firebase_config_path):
        with open(firebase_config_path, 'r') as f:
            config = json.load(f)
            public_dir = config.get('public', '')
            if public_dir:
                full_path = os.path.join(os.getcwd(), public_dir)
                if os.path.isdir(full_path):
                    root_path = full_path
                    print(f"Target static asset folder found via firebase.json: {public_dir}")

    # Generate Nginx configuration
    nginx_conf_path = os.path.join(layer_name, "nginx.conf")
    try:
        with open(nginx_conf_path, 'w') as f:
            # Add your Nginx configuration logic here
            f.write("Generated Nginx configuration for Firebase static site\n")
    except Exception as e:
        print(f"Error writing nginx.conf: {e}")
        sys.exit(1)

    # Setup entrypoint command
    print("Setup Nginx process to run with generated configuration")
    
if __name__ == "__main__":
    detect()
    build()
