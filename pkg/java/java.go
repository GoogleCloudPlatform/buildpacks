import os
import re
from typing import List, Optional

import requests
from googlecloudplatform.buildpacks import gcp, env, runtime, tooling

# Copyright 2021 Google LLC
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

class MavenProject:
    def __init__(self):
        self.plugins: List[MavenPlugin] = []
        self.profiles: List[MavenProfile] = []
        self.artifact_id: str = ""
        self.version: str = ""
        self.dependencies: List[MavenDependency] = []

class MavenProfile:
    def __init__(self):
        self.id: str = ""
        self.plugins: List[MavenPlugin] = []
        self.dependencies: List[MavenDependency] = []

class MavenPlugin:
    def __init__(self):
        self.group_id: str = ""
        self.artifact_id: str = ""
        self.configuration: MavenPluginConfiguration = MavenPluginConfiguration()

class MavenDependency:
    def __init__(self):
        self.group_id: str = ""
        self.artifact_id: str = ""

class MavenPluginConfiguration:
    def __init__(self):
        self.main_class: str = ""
        self.build_args: str = ""

def ParsePomFile(pom_file_content: bytes) -> Optional[MavenProject]:
    try:
        root = ET.fromstring(pom_file_content)
        
        project = MavenProject()
        project.artifact_id = root.findtext("artifactId") or ""
        project.version = root.findtext("version") or ""
        
        # Extract plugins
        plugins_section = root.findall("build/plugins/plugin")
        for plugin in plugins_section:
            m_plugin = MavenPlugin()
            m_plugin.group_id = plugin.findtext("groupId") or ""
            m_plugin.artifact_id = plugin.findtext("artifactId") or ""
            project.plugins.append(m_plugin)
            
        # Extract profiles
        profiles_section = root.findall("profiles/profile")
        for profile in profiles_section:
            m_profile = MavenProfile()
            m_profile.id = profile.findtext("id") or ""
            project.profiles.append(m_profile)
            
        return project
        
    except Exception as e:
        raise ValueError(f"Error parsing pom.xml: {e}")
