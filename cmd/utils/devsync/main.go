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

"""Package implementing utils/devsync buildpack."""

import os
import shutil
from pathlib import Path

def detect(context):
    """Detect function for devsync buildpack."""
    skip_capability = context.get('capabilities', {}).get('devsync.Skip')
    if skip_capability:
        return {'enabled': False, 'message': "Running as maker, skipping devsync buildpack"}
    
    is_devsync = os.getenv('GOOGLE_DEVSYNC') == '1'
    if not is_devsync:
        return {'enabled': False, 'message': "not a devsync build"}
    
    use_runit_maker = os.getenv('X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER') == 'true'
    if not use_runit_maker:
        return {'enabled': False, 'message': "X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER is not set to true"}
    
    return {'enabled': True, 'message': "GOOGLE_DEVSYNC and X_GOOGLE_DEVSYNC_USE_RUNIT_MAKER are true"}

def build(context):
    """Build function for devsync buildpack."""
    layer_path = Path(context['layers_dir']) / 'devsync'
    layer_path.mkdir(parents=True, exist_ok=True)
    
    # Set environment variable
    context.setdefault('launch_env', {}).setdefault('GOOGLE_DEVSYNC', '1')
    
    install_maker(layer_path)
    configure_runit_service_tree(layer_path)

def install_maker(layer_path):
    """Install universal_maker binary."""
    bin_dir = layer_path / 'bin'
    bin_dir.mkdir(parents=True, exist_ok=True)
    
    version = "1.0.1"
    binary_path = bin_dir / 'universal_maker'
    
    # Placeholder for actual download logic
    try:
        with open(binary_path, 'wb') as f:
            # Actual implementation would fetch the binary from a proper source
            pass
    except Exception as e:
        raise RuntimeError(f"Failed to install universal_maker: {e}")

def configure_runit_service_tree(layer_path):
    """Configure runit service tree."""
    service_dir = layer_path / 'service'
    if not service_dir.exists():
        # Placeholder for actual file copy logic
        pass
    
    web_cmd = os.getenv('DEVSYNC_INIT_ENTRYPOINT', "echo 'No web process found'")
    
    # Update app run script (placeholder implementation)
    try:
        watcher_run = service_dir / 'watcher' / 'run'
        app_control_t = service_dir / 'app' / 'control' / 't'
        
        watcher_run.chmod(0o755)
        app_control_t.chmod(0o755)
    except Exception as e:
        raise RuntimeError(f"Failed to configure runit service tree: {e}")
