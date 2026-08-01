"""
Package publisher provides basic functionality to coalesce user and framework adapter defined
variables.
"""

import logging
import os
from pathlib import Path
from typing import List, Optional

import yaml

from firebase_publisher import apphostingschema, bundleschema


class BuildSchema:
    def __init__(self):
        self.run_config: Optional[apphostingschema.RunConfig] = None
        self.env: List[apphostingschema.EnvironmentVariable] = []
        self.metadata: Optional[bundleschema.Metadata] = None

# Default values from GCP documentation
DEFAULT_CPU = 1
DEFAULT_MEMORY_MIB = 512
DEFAULT_CONCURRENCY = 80
DEFAULT_MAX_INSTANCES = 100
DEFAULT_MIN_INSTANCES = 0


def write_to_file(build_schema: BuildSchema, output_path: str) -> None:
    """
    Write the given build schema to the specified path.
    
    Args:
        build_schema: The build schema to serialize and write.
        output_path: Path where the YAML file will be written.
        
    Raises:
        IOError: If there's an error writing to the file.
    """
    try:
        # Convert BuildSchema to dict for serialization
        data = {
            "runConfig": build_schema.run_config.__dict__ if build_schema.run_config else None,
            "env": [vars(ev) for ev in build_schema.env],
            "metadata": build_schema.metadata.__dict__ if build_schema.metadata else None
        }
        
        file_data = yaml.safe_dump(data, default_flow_style=False)
        logging.info("Final build schema:\n%s\n. Note that any unset runConfig fields will be set to reasonable default values.", file_data)
        
        # Ensure parent directories exist
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        
        with open(output_path, 'w') as f:
            f.write(file_data)
            
    except IOError as e:
        raise IOError(f"Error writing to {output_path}: {e}") from e


def merge_environment_variables(
    aevs: List[apphostingschema.EnvironmentVariable],
    bevs: List[bundleschema.EnvironmentVariable]
) -> List[apphostingschema.EnvironmentVariable]:
    """
    Merge environment variables from apphosting and bundle schemas.
    
    Args:
        aevs: Environment variables from apphosting.yaml
        bevs: Environment variables from bundle.yaml
        
    Returns:
        Merged list of environment variables with precedence to apphosting variables.
    """
    merged = []
    var_by_name = {}
    
    for ev in aevs:
        var_by_name[ev.variable] = ev
    
    for ev in bevs:
        if ev.variable in var_by_name:
            logging.info("Using apphosting.yaml value/secret for environment variable %s", ev.variable)
        else:
            new_ev = apphostingschema.EnvironmentVariable(
                variable=ev.variable,
                value=ev.value,
                secret=ev.secret,
                availability=ev.availability
            )
            merged.append(new_ev)
    
    return aevs + merged


def merge_run_config(
    arc: Optional[apphostingschema.RunConfig],
    brc: Optional[bundleschema.RunConfig]
) -> apphostingschema.RunConfig:
    """
    Merge run configurations from apphosting and bundle schemas.
    
    Args:
        arc: Run config from apphosting.yaml
        brc: Run config from bundle.yaml
        
    Returns:
        Merged run configuration with precedence to apphosting values.
    """
    merged = apphostingschema.RunConfig()
    
    # Set defaults from bundle if not set in apphosting
    if arc is None and brc is not None:
        merged.cpu = brc.cpu
        merged.memory_mib = brc.memory_mib
        merged.concurrency = brc.concurrency
        merged.min_instances = brc.min_instances
        merged.max_instances = brc.max_instances
        merged.cpu_always_allocated = brc.cpu_always_allocated
    
    # Apply apphosting values on top of bundle
    if arc is not None:
        merged.cpu = arc.cpu or merged.cpu
        merged.memory_mib = arc.memory_mib or merged.memory_mib
        merged.concurrency = arc.concurrency or merged.concurrency
        merged.min_instances = arc.min_instances or merged.min_instances
        merged.max_instances = arc.max_instances or merged.max_instances
        merged.vpc_access = arc.vpc_access if arc.vpc_access else merged.vpc_access
        merged.cpu_always_allocated = arc.cpu_always_allocated if arc.cpu_always_allocated is not None else merged.cpu_always_allocated
    
    return merged


def to_build_schema(
    app_hosting_schema: apphostingschema.AppHostingSchema,
    bundle_schema: bundleschema.BundleSchema
) -> BuildSchema:
    """
    Merge apphosting and bundle schemas into a single build schema.
    
    Args:
        app_hosting_schema: Parsed apphosting.yaml configuration
        bundle_schema: Parsed bundle.yaml configuration
        
    Returns:
        Merged build schema containing all configurations.
    """
    build_schema = BuildSchema()
    
    # Merge run configurations
    merged_run_config = merge_run_config(app_hosting_schema.run_config, bundle_schema.run_config)
    build_schema.run_config = merged_run_config
    
    # Set metadata from bundle
    build_schema.metadata = bundle_schema.metadata
    
    # Merge environment variables
    merged_env = merge_environment_variables(
        app_hosting_schema.env,
        bundle_schema.run_config.environment_variables if bundle_schema.run_config else []
    )
    build_schema.env = merged_env
    
    return build_schema


def publish(app_hosting_path: str, bundle_path: str, output_path: str) -> None:
    """
    Merge configurations from apphosting.yaml and bundle.yaml into a single build schema.
    
    Args:
        app_hosting_path: Path to apphosting.yaml
        bundle_path: Path to bundle.yaml
        output_path: Where to write the merged configuration
    
    Raises:
        FileNotFoundError: If any input file is not found.
        yaml.YAMLError: If there's an error parsing YAML files.
    """
    try:
        # Read and validate schemas
        app_hosting_schema = apphostingschema.read_and_validate(Path(app_hosting_path))
        bundle_schema = bundleschema.read_and_validate(Path(bundle_path))
        
        # Merge into build schema
        build_schema = to_build_schema(app_hosting_schema, bundle_schema)
        
        # Write output
        write_to_file(build_schema, output_path)
        
    except FileNotFoundError as e:
        raise FileNotFoundError(f"Required file not found: {e.filename}") from e
    except yaml.YAMLError as e:
        raise yaml.YAMLError(f"Error parsing YAML file: {e}") from e
