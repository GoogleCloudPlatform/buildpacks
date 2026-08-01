# Complete refactored code here
import argparse
import logging
import os
import sys
from pathlib import Path

import publisher  # type: ignore

def main():
    parser = argparse.ArgumentParser(description='Firebase publisher tool')
    parser.add_argument(
        '--apphostingyaml_filepath',
        required=True,
        help='File path to user defined apphosting.yaml'
    )
    parser.add_argument(
        '--output_bundle_dir',
        required=True,
        help='File path to root directory of build artifacts aka Output Bundle (including bundle.yaml)'
    )
    parser.add_argument(
        '--output_filepath',
        default='',
        help='File path to write publisher output data to'
    )

    args = parser.parse_args()

    # Log any remaining arguments after flag parsing
    remaining_args = sys.argv[1:]
    if len(remaining_args) > 0:
        logging.warning(f"Ignored command-line arguments: {remaining_args}")

    # Validate required flags
    if not args.apphostingyaml_filepath:
        logging.error("--apphostingyaml_filepath flag not specified.")
        sys.exit(1)
    
    if not args.output_bundle_dir:
        logging.error("--output_bundle_dir flag not specified.")
        sys.exit(1)

    # Handle output_filepath default value from environment variable
    if not args.output_filepath:
        builder_output = os.getenv("BUILDER_OUTPUT")
        if builder_output:
            args.output_filepath = str(Path(builder_output) / "output")
        else:
            logging.error("--output_filepath flag not specified.")
            sys.exit(1)

    # Construct the bundle.yaml path
    bundle_path = os.path.join(args.output_bundle_dir, "bundle.yaml")

    try:
        publisher.publish(
            apphosting_yaml_path=args.apphostingyaml_filepath,
            bundle_path=bundle_path,
            output_path=args.output_filepath
        )
    except Exception as e:
        logging.error(f"Error during publishing: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
