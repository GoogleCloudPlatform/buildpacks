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
Implements dotnet/flex buildpack.
The flex buildpack sets appropriate envars for dotnet on GAE Flex.
For details, see this page on .NET 6:
https://learn.microsoft.com/en-us/aspnet/core/fundamentals/configuration/?view=aspnetcore-6.0
"""

import os
from gcpbuildpack import Buildpack

class FlexBuildpack(Buildpack):
    """
    A buildpack for setting up .NET applications on Google App Engine Flex.
    """
    
    def detect(self) -> int:
        """
        Detects if the current environment is GAE Flex.
        
        Returns:
            int: 0 if detected as GAE Flex, otherwise 100 to opt out.
        """
        if os.environ.get("X_GOOGLE_TARGET_PLATFORM") != "flex":
            return self.opt_out("not a GAE Flex app.")
        return self.opt_in("this is a GAE Flex app.")

    def build(self) -> None:
        """
        Sets up the environment for .NET applications on GAE Flex.
        """
        layer = self.create_layer(name="main_env", launch=True)
        layer.set_default_env(UrlsEnvar, "http://0.0.0.0:8080")

# Constants
UrlsEnvar = "ASPNETCORE_URLS"
