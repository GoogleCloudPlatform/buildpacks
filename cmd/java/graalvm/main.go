import logging
from pathlib import Path
import os
from typing import Dict, Any

import google_cloud_platform.buildpacks.gcp_buildpack as gcp
from google_cloud_platform.buildpacks.gcp_buildpack.types import (
    BuildPlanProvide,
    DetectResult,
)
from google_cloud_platform.buildpacks.env import is_using_native_image

logger = logging.getLogger(__name__)

# Constants
GRAALVM_VERSION = "21.0.0"
GRAALVM_URL_TEMPLATE = "https://github.com/graalvm/graalvm-ce-builds/releases/download/vm-{version}/graalvm-ce-java11-linux-amd64-{version}.tar.gz"
LAYER_NAME = "java-graalvm"
VERSION_KEY = "version"

# Provides
PROVIDES_GRAALVM = [BuildPlanProvide(name="graalvm")]
PLAN_PROVIDES = BuildPlan(Provides=PROVIDES_GRAALVM)

def detect_fn(context: gcp.Context) -> DetectResult:
    """Detect whether to opt-in or opt-out of the buildpack."""
    try:
        use_native_image, _ = is_using_native_image()
    except Exception as e:
        logger.error(f"Failed to parse GOOGLE_JAVA_USE_NATIVE_IMAGE: {e}")
        raise ValueError(f"failed to parse GOOGLE_JAVA_USE_NATIVE_IMAGE: {e}") from e

    if use_native_image:
        logger.warning("The GraalVM Native Image buildpack is enabled. Note: This is under development and not ready for use.")
        return gcp.OptInEnvSet(GOOGLE_JAVA_USE_NATIVE_IMAGE, plan_provides=PLAN_PROVIDES)
    
    return gcp.OptOutEnvNotSet(GOOGLE_JAVA_USE_NATIVE_IMAGE)

def build_fn(context: gcp.Context) -> None:
    """Build function that installs GraalVM and builds the native image."""
    install_graalvm(context)

class LayerContext:
    def __init__(self, layer_name: str):
        self.layer_name = layer_name
        self.metadata_path = os.path.join(layer_name, "metadata.json")

    def get_metadata(self) -> Dict[str, Any]:
        """Get metadata from the layer."""
        if not os.path.exists(self.metadata_path):
            return {}
        with open(self.metadata_path, 'r') as f:
            return json.load(f)

    def set_metadata(self, key: str, value: Any) -> None:
        """Set metadata for the layer."""
        current = self.get_metadata()
        current[key] = value
        os.makedirs(os.path.dirname(self.metadata_path), exist_ok=True)
        with open(self.metadata_path, 'w') as f:
            json.dump(current, f)

    def exec_command(self, command: str, user_attribution: bool = False) -> None:
        """Execute a shell command within the context."""
        logger.info(f"Executing command: {command}")
        subprocess.run(command, shell=True, check=True)

def install_graalvm(context: gcp.Context) -> None:
    """Install GraalVM into the specified layer."""
    try:
        graal_layer = context.Layer(LAYER_NAME)
    except Exception as e:
        logger.error(f"Failed to create {LAYER_NAME} layer: {e}")
        raise

    metadata = graal_layer.get_metadata()
    if metadata.get(VERSION_KEY) == GRAALVM_VERSION:
        logger.info("GraalVM cache hit, skipping installation.")
        return

    graal_layer.clear_layer()

    # Download and extract GraalVM
    archive_url = GRAALVM_URL_TEMPLATE.format(version=GRAALVM_VERSION)
    download_command = f"curl --fail --show-error --silent --location {archive_url} | tar xz --directory {graal_layer.path} --strip-components=1"
    graal_layer.exec_command(download_command)

    # Install native-image component
    graalUpdater = os.path.join(graal_layer.path, "bin", "gu")
    install_command = f"{graalUpdater} install native-image"
    graal_layer.exec_command(install_command)

    # Update metadata
    graal_layer.set_metadata(VERSION_KEY, GRAALVM_VERSION)
