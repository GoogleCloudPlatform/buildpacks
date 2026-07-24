# Complete refactored code here
import os
import json
import time
import random
import logging
from typing import Optional, Dict, Any
from pathlib import Path

import buildererror
import buildermetadata
import buildermetrics
import builderoutput

logger = logging.getLogger(__name__)

class Context:
    def __init__(self):
        self.buildpack_id = ""
        self.buildpack_version = ""
        self.warnings = []
        self.installed_runtime_versions = []
        self.stats = {
            "user": 0.0,
            "total": 0.0
        }

def save_error_output(ctx: Context, err: Exception):
    be = buildererror.Error.from_exception(err)
    output_dir = os.getenv(builderoutput.BUILDER_OUTPUT_ENV)
    if not output_dir:
        return

    max_message_bytes = 49000
    if len(be.message) > max_message_bytes:
        be.message = keep_tail(be.message, max_message_bytes)

    be.buildpack_id = ctx.buildpack_id
    be.buildpack_version = ctx.buildpack_version

    bo = builderoutput.BuilderOutput()
    bo.error = be
    bo.metrics = buildermetrics.global_metrics().to_dict()
    bo.metadata = buildermetadata.global_metadata().to_dict()

    try:
        data = json.dumps(bo.to_dict())
        Path(output_dir).mkdir(parents=True, exist_ok=True)
        temp_path = Path(output_dir) / f"{builderoutput.BUILDER_OUTPUT_FILENAME}-{random.randint(0, 1000)}"
        with open(temp_path, 'w') as f:
            f.write(data)
        final_path = Path(output_dir) / builderoutput.BUILDER_OUTPUT_FILENAME
        temp_path.rename(final_path)

    except Exception as e:
        logger.warning(f"Failed to save error output: {e}")

def keep_tail(message: str, max_length: int) -> str:
    if len(message) <= max_length:
        return message
    return f"...{message[-(max_length - 3):]}"

def keep_head(message: str, max_length: int) -> str:
    if len(message) <= max_length:
        return message
    return f"{message[:max_length-3]}..."

def save_success_output(ctx: Context, duration: float):
    output_dir = os.getenv(builderoutput.BUILDER_OUTPUT_ENV)
    if not output_dir:
        return

    bo = builderoutput.BuilderOutput()
    bo.installed_runtime_versions.extend(ctx.installed_runtime_versions)
    
    stats = {
        "buildpack_id": ctx.buildpack_id,
        "buildpack_version": ctx.buildpack_version,
        "duration_ms": duration * 1000,
        "user_duration_ms": ctx.stats["user"] * 1000
    }
    bo.stats.append(stats)
    
    try:
        data = json.dumps(bo.to_dict())
        Path(output_dir).mkdir(parents=True, exist_ok=True)
        with open(Path(output_dir) / builderoutput.BUILDER_OUTPUT_FILENAME, 'w') as f:
            f.write(data)
    except Exception as e:
        logger.warning(f"Failed to save success output: {e}")
