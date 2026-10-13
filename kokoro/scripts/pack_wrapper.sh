#!/bin/bash
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

# Wrapper around the pack binary that rewrites Maven and Gradle repository URLs
# in test application directories to use the GCS Maven Central mirror before
# invoking pack.

for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "--path" ]]; then
    next=$((i+1))
    src_dir="${!next}"
    if [[ -d "$src_dir" ]]; then
      python3 - "$src_dir" << 'PYEOF'
import os, sys

src_dir = sys.argv[1]
mirror = "https://maven-central.storage-download.googleapis.com/maven2/"
repo_xml = (
    "  <repositories>\n"
    "    <repository>\n"
    "      <id>central</id>\n"
    f"      <url>{mirror}</url>\n"
    "    </repository>\n"
    "  </repositories>\n"
)
plugin_repo_xml = (
    "  <pluginRepositories>\n"
    "    <pluginRepository>\n"
    "      <id>central</id>\n"
    f"      <url>{mirror}</url>\n"
    "    </pluginRepository>\n"
    "  </pluginRepositories>\n"
)

for root, _, files in os.walk(src_dir):
  for f in files:
    p = os.path.join(root, f)
    try:
      with open(p, "r", encoding="utf-8") as fh:
        content = fh.read()
    except Exception:
      continue
    orig = content
    if f == "pom.xml":
      content = content.replace("https://repo1.maven.org/maven2/", mirror)
      content = content.replace("https://repo1.maven.org/maven2", mirror.rstrip("/"))
      content = content.replace("https://repo.maven.apache.org/maven2/", mirror)
      content = content.replace("https://repo.maven.apache.org/maven2", mirror.rstrip("/"))
      if "<repositories>" not in content and "</project>" in content:
        content = content.replace("</project>", repo_xml + "</project>", 1)
      if "<pluginRepositories>" not in content and "</project>" in content:
        content = content.replace("</project>", plugin_repo_xml + "</project>", 1)
    elif f.endswith(".gradle"):
      content = content.replace('maven { url "https://jcenter.bintray.com" }', f'maven {{ url "{mirror}" }}')
      content = content.replace("mavenCentral()", f'maven {{ url "{mirror}" }}')
      content = content.replace("jcenter()", f'maven {{ url "{mirror}" }}')
    elif f.endswith(".gradle.kts"):
      content = content.replace("mavenCentral()", f'maven {{ url = uri("{mirror}") }}')
      content = content.replace("jcenter()", f'maven {{ url = uri("{mirror}") }}')
    elif f in ("mvnw", "mvnw.cmd", "maven-wrapper.properties", "MavenWrapperDownloader.java"):
      content = content.replace("https://repo.maven.apache.org/maven2", mirror.rstrip("/"))
      content = content.replace("https://repo1.maven.org/maven2", mirror.rstrip("/"))
    if content != orig:
      st = os.stat(p)
      os.chmod(p, (st.st_mode & 0o777) | 0o644)
      with open(p, "w", encoding="utf-8") as fh:
        fh.write(content)
PYEOF
    fi
  fi
done
exec "$(dirname "$0")/pack.bin" "$@"
