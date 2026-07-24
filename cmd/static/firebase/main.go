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

"""
Implements static/firebase buildpack.
The static firebase buildpack detects firebase.json with web assets, generates a default
SPA-friendly nginx configuration, and adds the web startup process.
"""

import logging
from pathlib import Path
from typing import Dict, Any

from google.cloud import buildpacks  # type: ignore
import jinja2  # type: ignore

logger = logging.getLogger(__name__)

class BuildPack:
    @staticmethod
    def detect() -> Dict[str, Any]:
        """Detects if the Firebase buildpack should be applied."""
        firebase_json = Path("firebase.json")
        if not firebase_json.exists():
            return {}

        result = {
            "buildpack": {
                "id": "google.firebase",
                "version": "1.0.0"
            },
            "config_files": ["firebase.json"]
        }
        logger.info("Firebase project detected")
        return result

    @staticmethod
    def build() -> None:
        """Performs the build steps for the Firebase application."""
        # Generate default nginx configuration if not exists
        nginx_conf = Path("nginx.conf")
        if not nginx_conf.exists():
            template_loader = jinja2.FileSystemLoader(searchpath="./templates")
            template_env = jinja2.Environment(loader=template_loader)
            template = template_env.get_template("nginx.conf.j2")

            with open(nginx_conf, "w") as f:
                f.write(template.render())
            logger.info("Generated default nginx configuration")

        # Install node.js if necessary
        package_json = Path("package.json")
        if package_json.exists():
            logger.info("Installing Node.js dependencies")
            # Add logic to install dependencies here
            pass

def main() -> None:
    """Main entry point for the Firebase buildpack."""
    gcp_main = buildpacks.GCPBuildPack()
    gcp_main.run(BuildPack.detect, BuildPack.build)

if __name__ == "__main__":
    main()
