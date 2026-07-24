import argparse
import asyncio
import logging
import os
from pydantic import BaseModel, ValidationError

class PublishInput(BaseModel):
    apphostingyaml_filepath: str
    output_bundle_dir: str
    output_filepath: str

async def publish(args: PublishInput) -> None:
    try:
        from firebase.publisher import Publisher
        publisher = Publisher()
        await publisher.Publish(
            args.apphostingyaml_filepath,
            args.output_bundle_dir,
            args.output_filepath
        )
    except Exception as e:
        logging.error(f"Publish failed: {e}")
        raise

def main():
    parser = argparse.ArgumentParser(description="Firebase publisher")
    parser.add_argument("--apphostingyaml_filepath", required=True, help="File path to user defined apphosting.yaml")
    parser.add_argument("--output_bundle_dir", required=True, help="File path to root directory of build artifacts (including bundle.yaml)")
    parser.add_argument("--output_filepath", default=None, help="File path to write publisher output data to")

    args = parser.parse_args()

    if args.output_filepath is None:
        builder_output = os.getenv("BUILDER_OUTPUT")
        if builder_output:
            args.output_filepath = os.path.join(builder_output, "output")
        else:
            parser.error("--output_filepath must be specified or BUILDER_OUTPUT environment variable set.")

    try:
        publish_input = PublishInput(**vars(args))
    except ValidationError as e:
        print(f"Validation error: {e}")
        return

    asyncio.run(publish(publish_input))

if __name__ == "__main__":
    main()
