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

"""
Implements dotnet/publish buildpack.
The publish buildpack runs dotnet publish.
"""

import argparse
import logging
import os

from google.cloud import buildpacks_v2beta1

class Buildpack:
    def __init__(self):
        self.detector = lib.Detector()
        self.builder = lib.Builder()

    def detect(self, context_dir: str) -> bool:
        """
        Detect if the current directory is a .NET project that requires publishing.

        Args:
            context_dir (str): The directory to check for .NET files.

        Returns:
            bool: True if it's a .NET project requiring publish, False otherwise.
        """
        return self.detector.detect(context_dir)

    def build(self, context_dir: str) -> None:
        """
        Publish the .NET project in the specified directory.

        Args:
            context_dir (str): The directory containing the .NET project to publish.
        """
        logging.info("Starting .NET publish process...")
        self.builder.publish(context_dir)
        logging.info("Publish completed successfully.")

def main():
    parser = argparse.ArgumentParser(description='Run .NET publish buildpack.')
    parser.add_argument('--context', required=True,
                       help='The directory containing the .NET project to publish.')

    args = parser.parse_args()

    buildpack = Buildpack()

    if not buildpack.detect(args.context):
        logging.info("No .NET project found, exiting.")
        return

    buildpack.build(args.context)

if __name__ == '__main__':
    main()
