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
Firebase Static Buildpack

Implements a Firebase static site buildpack that detects firebase.json with web assets,
generates a default SPA-friendly nginx configuration, and adds the web startup process.
"""

import json
import os
from pathlib import Path
import shutil
from typing import Dict, Optional

import click
from fastapi import FastAPI
from pydantic import BaseModel, Field

app = FastAPI()

class FirebaseConfig(BaseModel):
    hosting: dict = Field(..., description="Firebase Hosting configuration")
    public: str = Field(..., description="Public directory path")
    rewrites: list = Field(..., description="Rewrite rules for the application")

async def detect() -> bool:
    """
    Detects if firebase.json exists and contains valid Firebase Hosting configuration.

    Returns:
        bool: True if Firebase Hosting is detected, False otherwise.
    """
    try:
        firebase_config_path = Path("firebase.json")
        if not firebase_config_path.exists():
            return False

        with open(firebase_config_path, "r") as f:
            config = json.load(f)

        # Validate required fields
        if "hosting" not in config or "public" not in config["hosting"]:
            return False

        FirebaseConfig(**config["hosting"])
        return True

    except Exception as e:
        print(f"Error detecting Firebase Hosting configuration: {e}")
        return False

async def build() -> None:
    """
    Generates the necessary nginx configuration for serving a Firebase static site.
    """
    try:
        # Read firebase.json
        firebase_config_path = Path("firebase.json")
        with open(firebase_config_path, "r") as f:
            config = json.load(f)

        firebase_hosting = FirebaseConfig(**config["hosting"])

        # Generate nginx configuration
        nginx_conf = f"""
        server {{
            listen 80 default_server;
            listen [::]:80 default_server;
            server_name _;
            root {firebase_hosting.public};
            index index.html;

            location / {{
                try_files $uri $uri/ /index.html;
            }}
        }}
        """

        # Write nginx configuration to file
        output_path = Path("nginx.conf")
        with open(output_path, "w") as f:
            f.write(nginx_conf.strip())

    except Exception as e:
        print(f"Error building Firebase Hosting configuration: {e}")
        raise

@app.get("/health")
async def health_check() -> Dict[str, str]:
    """
    Health check endpoint.

    Returns:
        Dict[str, str]: Health status
    """
    return {"status": "ok"}

@click.command()
@click.option("--detect", is_flag=True, help="Detect Firebase Hosting configuration.")
@click.option("--build", is_flag=True, help="Build Firebase Hosting configuration.")
async def main(detect: bool = False, build: bool = False) -> None:
    """
    Main entry point for the Firebase Static Buildpack.

    Args:
        detect (bool): Whether to run detection only.
        build (bool): Whether to run build only.
    """
    if detect and not build:
        print("Detecting Firebase Hosting configuration...")
        result = await detect()
        print(f"Firebase Hosting detected: {result}")

    elif build and not detect:
        print("Building Firebase Hosting configuration...")
        await build()

    else:
        print("Either --detect or --build must be specified.")

if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
