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
Implements Ruby Functions Framework buildpack.
The functions_framework buildpack sets up the execution environment to
run the Ruby Functions Framework. The framework itself, with its converter,
is always installed as a dependency.
"""

import logging

from google.cloud.functions import BuildpackBuilder
from .lib import detect_ruby_functions_framework, build_ruby_functions_framework

logger = logging.getLogger(__name__)

class RubyFunctionsFrameworkBuilder(BuildpackBuilder):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    async def detect(self):
        """
        Detects if the current environment requires Ruby Functions Framework.
        Returns True if applicable, False otherwise.
        """
        return await detect_ruby_functions_framework()

    async def build(self):
        """
        Builds and sets up the Ruby Functions Framework execution environment.
        """
        logger.info("Setting up Ruby Functions Framework environment")
        await build_ruby_functions_framework()

if __name__ == "__main__":
    # Initialize and run the builder
    builder = RubyFunctionsFrameworkBuilder()
    try:
        if asyncio.run(builder.detect()):
            asyncio.run(builder.build())
    except Exception as e:
        logger.error(f"Error running buildpack: {e}")
        raise
