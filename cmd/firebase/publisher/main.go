# Copyright 2023 Google LLC
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
import logging
import os
import sys
from pathlib import Path

import publisher  # Assuming a corresponding Python package structure

def main():
    parser = argparse.ArgumentParser(description='Run AppHosting publisher logic.')
    parser.add_argument('--apphostingyaml_filepath', type=str, required=True,
                       help='File path to user defined apphosting.yaml')
    parser.add_argument('--output_bundle_dir', type=str, required=True,
                       help='File path to root directory of build artifacts aka Output Bundle (including bundle.yaml)')
    parser.add_argument('--output_filepath', type=str,
                       help='File path to write publisher output data to')

    args = parser.parse_args()

    # Log any remaining arguments
    if len(sys.argv) > 1:
        _, *remaining_args = sys.argv[1:]
        if remaining_args:
            logging.warning(f"Ignored command-line arguments: {remaining_args}")

    # Validate required arguments and set defaults where necessary
    apphosting_yaml_path = args.apphostingyaml_filepath
    output_bundle_dir = args.output_bundle_dir

    if not apphosting_yaml_path:
        parser.error("--apphostingyaml_filepath flag not specified.")

    if not output_bundle_dir:
        parser.error("--output_bundle_dir flag not specified.")

    # Handle output_filepath with environment variable fallback
    output_file_path = args.output_filepath
    if not output_file_path:
        builder_output = os.getenv('BUILDER_OUTPUT')
        if builder_output:
            output_file_path = os.path.join(builder_output, 'output')
        else:
            parser.error("--output_filepath flag not specified.")

    # Construct the paths
    bundle_yaml_path = os.path.join(output_bundle_dir, 'bundle.yaml')

    try:
        publisher.publish(
            apphosting_yaml_path,
            bundle_yaml_path,
            output_file_path
        )
    except Exception as e:
        logging.error(f"Publisher error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
