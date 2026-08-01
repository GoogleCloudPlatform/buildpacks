# Complete refactored code here
import os
import yaml
from dataclasses import dataclass, field
from typing import List, Optional, Dict

from google.cloud import apphostingschema  # Assuming this is the correct import path


class EnvironmentVariable:
    def __init__(self, variable: str, value: str, availability: List[str], source: str = apphostingschema.SourceFirebaseSystem):
        self.variable = variable
        self.value = value
        self.availability = availability
        self.source = source

    def validate(self) -> None:
        if not self.value or self.secret:
            raise ValueError(f"For environment variable {self.variable}, 'value' is required and 'secret' should not be present")

        valid_availabilities = {"RUNTIME"}
        for avail in self.availability:
            if avail not in valid_availabilities:
                raise ValueError(f"Invalid value {avail} in 'availability'")


@dataclass
class VpcAccess:
    connector: str


@dataclass
class RunConfig:
    environment_variables: List[EnvironmentVariable] = field(default_factory=list)
    cpu: Optional[float] = None
    memory_mib: Optional[int] = None
    concurrency: Optional[int] = None
    max_instances: Optional[int] = None
    min_instances: Optional[int] = None
    vpc_access: Optional[VpcAccess] = None
    cpu_always_allocated: Optional[bool] = None

    def validate(self) -> None:
        for env_var in self.environment_variables:
            env_var.validate()


@dataclass
class Metadata:
    adapter_package_name: str
    adapter_version: str
    framework: str
    framework_version: str

    def validate(self) -> None:
        if not self.adapter_package_name:
            raise ValueError("Missing the adapter package name in bundle.yaml metadata")
        if not self.adapter_version:
            raise ValueError("Missing the adapter version in bundle.yaml metadata")
        if not self.framework:
            raise ValueError("Missing the framework name in bundle.yaml metadata")
        if not self.framework_version:
            raise ValueError("Missing the framework version in bundle.yaml metadata")


@dataclass
class BundleSchema:
    run_config: RunConfig = field(default_factory=RunConfig)
    metadata: Optional[Metadata] = None

    def validate(self) -> None:
        self.run_config.validate()
        if self.metadata is not None:
            self.metadata.validate()


def read_and_validate_from_file(file_path: str) -> BundleSchema:
    try:
        with open(file_path, 'r') as f:
            data = yaml.safe_load(f)
    except FileNotFoundError:
        raise ValueError(f"Missing output bundle config at {file_path}")
    except Exception as e:
        raise ValueError(f"Reading output bundle config at {file_path}: {e}")

    # Parse the data into BundleSchema
    run_config_data = data.get('runConfig', {})
    metadata_data = data.get('metadata')

    # Process RunConfig
    environment_vars = []
    if 'environmentVariables' in run_config_data:
        for env_var_dict in run_config_data['environmentVariables']:
            variable = env_var_dict.get('variable', '')
            value = env_var_dict.get('value', '')
            availability = env_var_dict.get('availability', [])
            source = env_var_dict.get('source', apphostingschema.SourceFirebaseSystem)
            env_var = EnvironmentVariable(variable, value, availability, source)
            environment_vars.append(env_var)

    run_config = RunConfig(
        environment_variables=environment_vars,
        cpu=run_config_data.get('cpu'),
        memory_mib=run_config_data.get('memoryMiB'),
        concurrency=run_config_data.get('concurrency'),
        max_instances=run_config_data.get('maxInstances'),
        min_instances=run_config_data.get('minInstances'),
        vpc_access=VpcAccess(run_config_data['vpcAccess']['connector']) if 'vpcAccess' in run_config_data else None,
        cpu_always_allocated=run_config_data.get('cpuAlwaysAllocated')
    )

    # Process Metadata
    metadata = None
    if metadata_data:
        metadata = Metadata(
            adapter_package_name=metadata_data['adapterPackageName'],
            adapter_version=metadata_data['adapterVersion'],
            framework=metadata_data['framework'],
            framework_version=metadata_data['frameworkVersion']
        )

    bundle_schema = BundleSchema(run_config, metadata)
    bundle_schema.validate()

    return bundle_schema
