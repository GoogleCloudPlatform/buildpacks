from dataclasses import dataclass
import json
from pathlib import Path
from typing import Optional
import semver

@dataclass
class PackageJSON:
    engines: dict[str, str]
    package_manager: str
    dev_dependencies: dict[str, str]

class PNPMLayer:
    def __init__(self, name: str):
        self.name = name
        self.path = Path(name)
        self.metadata = {}

class PNPMInstaller:
    def install_pnpm(self, context: dict, layer: PNPMLayer, package_json: PackageJSON) -> None:
        raise NotImplementedError

def detect_pnpm_version(context: dict, package_json: PackageJSON, stack_id: str) -> str:
    # Implementation logic here
    return "12.3.4"  # Placeholder

def install_pnpm(context: dict, layer: PNPMLayer, package_json: PackageJSON) -> None:
    if context.get("disabled_capabilities", {}).get("nodejs.PnpmInstaller"):
        print("PNPMInstaller capability is disabled. Skipping installation.")
        return

    capability = context.get("capabilities", {}).get("nodejs.PnpmInstaller")
    if capability:
        installer = capability()
        if isinstance(installer, PNPMInstaller):
            installer.install_pnpm(context, layer, package_json)
            return

    # Default implementation
    pass  # Remaining logic here
