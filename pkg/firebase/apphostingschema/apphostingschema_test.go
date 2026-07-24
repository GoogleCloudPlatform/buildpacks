import os
import re
from dataclasses import dataclass, field
from typing import List, Optional
import yaml
from .vpcaccess import VpcAccess, validate_vpc_access

reserved.FirebaseKeyPrefix = "X_FIREBASE_"

@dataclass
class EnvironmentVariable:
    Variable: str
    Value: Optional[str] = None
    Secret: Optional[str] = None
    Availability: List[str] = field(default_factory=list)
    Source: str = ""
    
    def __post_init__(self):
        if self.Value and self.Secret:
            raise ValueError("Both 'value' and 'secret' cannot be present")
        if not self.Value and not self.Secret:
            raise ValueError("Either 'value' or 'secret' must be present")
        for avail in self.Availability:
            if avail not in valid_availability_values:
                raise ValueError(f"Invalid availability value: {avail}")

@dataclass
class RunConfig:
    CPU: Optional[float] = None
    MemoryMiB: Optional[int] = None
    Concurrency: Optional[int] = None
    MaxInstances: Optional[int] = None
    MinInstances: Optional[int] = None
    VpcAccess: Optional[VpcAccess] = None
    CPUAlwaysAllocated: Optional[bool] = None

@dataclass
class Scripts:
    RunCommand: str = ""
    BuildCommand: str = ""

@dataclass
class OutputFiles:
    ServerApp: 'ServerApp'

@dataclass
class ServerApp:
    Include: List[str]

@dataclass
class AppHostingSchema:
    RunConfig: Optional[RunConfig] = None
    Env: List[EnvironmentVariable] = field(default_factory=list)
    Scripts: Optional[Scripts] = None
    OutputFiles: Optional[OutputFiles] = None

def read_and_validate_from_file(file_path: str) -> 'AppHostingSchema':
    if not os.path.exists(file_path):
        return AppHostingSchema()
    
    with open(file_path, 'r') as f:
        data = yaml.safe_load(f)
        
    try:
        return AppHostingSchema.from_yaml(data)
    except Exception as e:
        raise ValueError(f"Invalid apphosting.yaml at {file_path}: {e}")

@classmethod
def from_yaml(cls, data: dict) -> 'AppHostingSchema':
    run_config_data = data.get('runConfig')
    run_config = RunConfig() if run_config_data else None
    # Populate run_config fields...
    
    env_vars = [EnvironmentVariable(**ev) for ev in data.get('env', [])]
    
    scripts_data = data.get('scripts')
    scripts = Scripts(**scripts_data) if scripts_data else None
    
    output_files_data = data.get('outputFiles')
    output_files = OutputFiles(ServerApp(**output_files_data['serverApp'])) if output_files_data else None
    
    return cls(RunConfig=run_config, Env=env_vars, Scripts=scripts, OutputFiles=output_files)

def is_reserved_key(env_key: str) -> bool:
    return env_key in reserved_keys or env_key.startswith(reserved.FirebaseKeyPrefix)

def sanitize_env(env_vars: List[EnvironmentVariable]) -> List[EnvironmentVariable]:
    sanitized = []
    for ev in env_vars:
        if not is_reserved_key(ev.Variable):
            if not ev.Availability:
                ev.Availability = ["BUILD", "RUNTIME"]
                print(f"INFO: {ev.Variable} has no availability specified, defaulting to ['BUILD', 'RUNTIME']")
            sanitized.append(ev)
        else:
            print(f"WARNING: Reserved key {ev.Variable} removed from environment variables")
    return sanitized

def sanitize(schema: 'AppHostingSchema') -> None:
    schema.Env = sanitize_env(schema.Env)

def merge_app_hosting_schemas(base_schema: 'AppHostingSchema', env_specific_schema: 'AppHostingSchema') -> None:
    # Merge RunConfig fields...
    if env_specific_schema.RunConfig:
        base_schema.RunConfig = env_specific_schema.RunConfig
        
    # Merge Env variables...
    merged_env = []
    env_vars_by_name = {ev.Variable: ev for ev in env_specific_schema.Env}
    for ev in base_schema.Env:
        if ev.Variable not in env_vars_by_name:
            merged_env.append(ev)
    merged_env.extend(env_specific_schema.Env)
    base_schema.Env = merged_env
    
    # Merge other sections...
    if env_specific_schema.OutputFiles:
        base_schema.OutputFiles = env_specific_schema.OutputFiles

def merge_with_environment_specific_yaml(base_schema: 'AppHostingSchema', apphosting_path: str, environment_name: str) -> None:
    if not environment_name:
        return
        
    env_file_path = os.path.join(os.path.dirname(apphosting_path), f"apphosting.{environment_name}.yaml")
    if not os.path.exists(env_file_path):
        print(f"INFO: Environment specific file {env_file_path} not found")
        return
        
    env_specific_schema = read_and_validate_from_file(env_file_path)
    merge_app_hosting_schemas(base_schema, env_specific_schema)

def get_env_var(schema: 'AppHostingSchema', key: str) -> Optional[EnvironmentVariable]:
    for ev in schema.Env:
        if ev.Variable == key:
            return ev
    return None

def write_to_file(self, file_path: str) -> None:
    with open(file_path, 'w') as f:
        yaml.dump(self.to_dict(), f)

def to_dict(self) -> dict:
    return {
        'runConfig': self.RunConfig.__dict__ if self.RunConfig else None,
        'env': [ev.__dict__ for ev in self.Env],
        'scripts': self.Scripts.__dict__ if self.Scripts else None,
        'outputFiles': self.OutputFiles.__dict__ if self.OutputFiles else None
    }
