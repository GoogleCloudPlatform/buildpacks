# Copyright 2025 Google LLC
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

"""Implements ruby entrypoint buildpack for flex."""

import os
import logging

from typing import Optional, Dict, Any

FLEX_ENTRYPOINT = "flex_entrypoint"
PRODUCTION_ENV = "production"

def detect_fn(context: Dict[str, Any]) -> bool:
    """Detects if this is a GAE Flex application."""
    # Check if it's a Flex environment
    if not is_flex():
        logging.info("Not a GAE Flex app.")
        return False

    # Check if entrypoint is set in environment variables
    entrypoint = os.getenv(env.ENTRYPOINT, "")
    if entrypoint:
        logging.info("Entrypoint found in environment variables.")
        return True

    # Try to get entrypoint from app.yaml
    try:
        entrypoint_from_yaml = appyaml.get_entrypoint(context["application_root"])
        if entrypoint_from_yaml:
            logging.info("Using entrypoint from app.yaml.")
            return True
    except Exception as e:
        logging.error(f"Error finding entrypoint in app.yaml: {e}")
        return False

    # If none of the above, opt out
    logging.info("No valid entrypoint found.")
    return False

def build_fn(context: Dict[str, Any]) -> None:
    """Builds and configures the application."""
    entrypoint = get_entrypoint(context)
    if not entrypoint:
        logging.error("Could not determine entrypoint.")
        return

    # Configure launch environment
    context["layers"][FLEX_ENTRYPOINT] = {
        "launch_env": {
            "RACK_ENV": PRODUCTION_ENV,
            "RAILS_ENV": PRODUCTION_ENV,
            "APP_ENV": PRODUCTION_ENV
        }
    }

    logging.info(f"Using entrypoint: {entrypoint}")
    context["processes"] = [{
        "name": "web",
        "command": [entrypoint],
        "args": []
    }]

def get_entrypoint(context: Dict[str, Any]) -> str:
    """Determines the application's entrypoint."""
    # Check environment variable first
    entrypoint = os.getenv(env.ENTRYPOINT, "")
    if entrypoint:
        return entrypoint

    # Try to get from app.yaml
    try:
        entrypoint_from_yaml = appyaml.get_entrypoint(context["application_root"])
        if entrypoint_from_yaml:
            return entrypoint_from_yaml
    except Exception as e:
        logging.warning(f"app.yaml env var set but file doesn't exist: {e}")
        return ""

    # Infer from Ruby context
    try:
        inferred_ep = ruby.infer_entrypoint(context, context["application_root"])
        return inferred_ep
    except Exception as e:
        logging.error(e)
        return ""
