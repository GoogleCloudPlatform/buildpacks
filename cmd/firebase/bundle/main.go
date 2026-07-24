import logging
import os
import sys
from typing import Optional
import asyncio
from pydantic import BaseSettings
from fastapi.logger import logger

class FirebaseBuildConfig(BaseSettings):
    project_id: str
    output_dir: str = "firebase_output"
    static_assets_dir: str = "public"

    class Config:
        env_file = ".env"

async def detect() -> bool:
    """Detect if Firebase buildpack should be applied."""
    try:
        # Check for Firebase usage indicators (e.g., presence of firebase.json)
        config = FirebaseBuildConfig()
        logger.info("Checking for Firebase project configuration...")
        return os.path.exists("firebase.json")
    except Exception as e:
        logger.error(f"Error during detection: {str(e)}")
        return False

async def build() -> None:
    """Perform the Firebase bundle build operations."""
    try:
        config = FirebaseBuildConfig()

        # 1. Copy static assets
        source_dir = os.path.join(os.getcwd(), config.static_assets_dir)
        dest_dir = os.path.join(config.output_dir, "static")

        logger.info(f"Copying static assets from {source_dir} to {dest_dir}")
        await asyncio.to_thread(
            shutil.copytree,
            source_dir,
            dest_dir,
            dirs_exist_ok=True
        )

        # 2. Override run script
        run_script_content = (
            "#!/bin/bash\n"
            f"firebase serve --project {config.project_id}"
        )

        run_script_path = os.path.join(config.output_dir, "run")
        logger.info(f"Creating new run script at {run_script_path}")

        async with asyncio.open_asyncio(run_script_path, mode='w') as f:
            await f.write(run_script_content)

        # Make run script executable
        await asyncio.to_thread(
            os.chmod,
            run_script_path,
            0o755
        )

        logger.info("Firebase bundle build completed successfully.")
    except Exception as e:
        logger.error(f"Error during build: {str(e)}")
        sys.exit(1)

async def main():
    """Main entry point for the Firebase bundle buildpack."""
    if await detect():
        await build()
    else:
        logger.info("Firebase buildpack not applicable. Skipping.")
        sys.exit(0)

def run_main():
    """Synchronous wrapper to run the async main function."""
    asyncio.run(main())

if __name__ == "__main__":
    run_main()
