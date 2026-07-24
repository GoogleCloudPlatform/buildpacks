"""
Package execd provides an interface for installing exec.d scripts.
"""

from abc import ABC, abstractmethod
from typing import Any


# Note: In a full repository migration, Context would be imported from the 
# migrated gcpbuildpack package. For the purpose of this module's definition:
class Context:
    """Placeholder for gcp.Context."""
    pass


# INSTALLER_CAPABILITY is the key used to inject the ExecDInstaller capability.
INSTALLER_CAPABILITY = "execd.Installer"


class Installer(ABC):
    """
    Defines the interface for installing exec.d scripts.
    This allows abstracting the installation logic so that it can be swapped out
    for the 'maker' use case, avoiding file access to missing buildpack directories.
    """

    @abstractmethod
    def install(self, ctx: Context, layer_name: str, script_rel_path: str) -> None:
        """
        Installs an exec.d script.

        Args:
            ctx: The GCP context.
            layer_name: The name of the layer.
            script_rel_path: The relative path to the script.

        Raises:
            Exception: If installation fails.
        """
        pass


class MakerInstaller(Installer):
    """
    Implements the Installer interface for the maker tool.
    It performs a no-op instead of trying to copy scripts from the buildpack root,
    which is not available in the maker environment.
    """

    def install(self, ctx: Context, layer_name: str, script_rel_path: str) -> None:
        """
        Does nothing, avoiding the 'file not found' error in maker mode.
        """
        # No-op for maker.
        return
