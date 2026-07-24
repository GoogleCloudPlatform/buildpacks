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

import argparse
import os

def extract_package(source):
    """Extracts the Go package name from a directory of .go files."""
    if not os.path.isdir(source):
        return None, f"Directory does not exist: {source}"
    
    package_name = None
    go_files = [f for f in os.listdir(source) if f.endswith('.go')]
    
    if not go_files:
        return None, "No .go files found in the directory."
    
    for filename in go_files:
        file_path = os.path.join(source, filename)
        with open(file_path, 'r') as f:
            package_found = False
            for line in f:
                stripped_line = line.strip()
                if stripped_line.startswith('package'):
                    parts = stripped_line.split()
                    current_pkg = parts[1].rstrip(';')
                    package_found = True
                    break
            if not package_found:
                return None, f"No package declaration found in {filename}"
            
            if package_name is None:
                package_name = current_pkg
            elif current_pkg != package_name:
                return None, (f"Multiple packages detected: {package_name} vs. "
                              f"{current_pkg} in {filename}")
    
    return package_name, None

def main():
    """Main entry point for the get_package script."""
    parser = argparse.ArgumentParser(description='Extract Go package name from a directory.')
    parser.add_argument('--dir', required=True, help='Directory containing *.go files')
    args = parser.parse_args()

    package, err = extract_package(args.dir)
    if err is not None:
        print(f"Error: {err}")
        exit(1)
    
    print(package)

if __name__ == "__main__":
    main()
