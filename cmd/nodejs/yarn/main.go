"""Implements nodejs/yarn buildpack.
The npm buildpack installs dependencies using yarn and installs yarn itself if not present.
"""

import sys
import os
import asyncio
from typing import Any
from pydantic import BaseModel, BaseSettings, Field
from fastapi import FastAPI

# Assuming these are the converted async functions from the original Go library
from .lib import detect, build  # type: ignore

class Settings(BaseSettings):
    """Pydantic settings model for environment variables."""

    project_id: str = Field(..., env="GOOGLE_CLOUD_PROJECT")
    region: str = Field("us-central1", env="GOOGLE_CLOUD_REGION")

    class Config:
        """Configuration for the settings model."""

        env_file = ".env"

async def main() -> None:
    """Main entry point for the buildpack."""

    try:
        # Parse command-line arguments
        args = sys.argv[1:]
        if not args:
            print("Error: No arguments provided")
            return

        settings = Settings()
        remaining_args = await detect(settings=settings, args=args)
        await build(settings=settings, args=remaining_args)

    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())
