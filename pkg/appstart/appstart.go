# Complete refactored code here
import json
import os
from enum import Enum
from dataclasses import dataclass
from typing import Any, Optional

gcp = __import__('github.com/GoogleCloudPlatform/buildpacks.pkg.gcpbuildpack')

class EntrypointType(Enum):
    Default = 0
    Generated = 1
    User = 2
    
    def __str__(self) -> str:
        return self.name.capitalize()

@dataclass
class Entrypoint:
    Type: Optional[EntrypointType] = None
    Command: Optional[str] = None
    WorkDir: Optional[str] = None

@dataclass
class Config:
    Runtime: Optional[str] = None
    Entrypoint: Optional[Entrypoint] = None
    MainExecutable: Optional[str] = None

ConfigDir = ".googleconfig"
configFile = f"{ConfigDir}/app_start.json"

def write_config(config: Config, ctx: gcp.Context) -> None:
    layer_name = "config"
    layer_type = gcp.LaunchLayer
    
    try:
        # Create or get the layer
        l = ctx.layer(layer_name, layer_type)
        
        # Marshal the config to JSON
        json_content = json.dumps(config.__dict__)
        
        # Remove existing config directory if it exists
        ctx.remove_all(ConfigDir)
        
        # Symlink layer path to config dir
        ctx.symlink(l.path, ConfigDir)
        
        # Write the config file with appropriate permissions
        mode = 0o444
        ctx.write_file(configFile, json_content, mode)
        
    except Exception as e:
        raise RuntimeError(f"Error writing app_start.json: {e}") from e

def main():
    pass  # Add any necessary main logic here if needed

if __name__ == "__main__":
    main()
