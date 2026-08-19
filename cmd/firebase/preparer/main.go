import argparse
import asyncio
import logging
import os
from contextlib import AsyncExitStack

from google.cloud.secretmanager_v1 import SecretManagerServiceClient
from google.api_core.exceptions import GoogleAPICallError

from pkg.firebase.util.filesystem import detect_app_hosting_yaml_path
from pkg.firebase.preparer import prepare

def main():
    parser = argparse.ArgumentParser(description='Run preprocessing steps for App Hosting backend builds.')
    parser.add_argument('--apphostingyaml_filepath', type=str, help='File path to user defined apphosting.yaml')
    parser.add_argument('--workspace_path', type=str, default='/workspace', help='File path to the workspace directory')
    parser.add_argument('--project_id', required=True, type=str, help='User\'s GCP project ID')
    parser.add_argument('--region', type=str, help='Current GCP Region. Used to expand resource IDs.')
    parser.add_argument('--environment_name', type=str, help='Environment name tied to the build, if applicable')
    parser.add_argument('--apphostingyaml_output_filepath', required=True, type=str, help='File path to write the validated and formatted apphosting.yaml to')
    parser.add_argument('--apphosting_preprocessed_path_for_pack', type=str, default='/workspace/apphosting_preprocessed',
                      help='File path to write the preprocessed apphosting.yaml to for pack step to consume')
    parser.add_argument('--dot_env_output_filepath', required=True, type=str, help='File path to write the output .env file to')
    parser.add_argument('--backend_root_directory', required=True, type=str, help='File path to the application directory specified by the user')
    parser.add_argument('--buildpack_config_output_filepath', required=True, type=str, help='File path to write the buildpack config to')
    parser.add_argument('--firebase_config', type=str, help='JSON serialized Firebase config used by Firebase Admin SDK')
    parser.add_argument('--firebase_webapp_config', type=str, help='JSON serialized Firebase config used by Firebase Client SDK')
    parser.add_argument('--server_side_env_vars', type=str, help='List of server side env vars to set. An empty string indicates server side environment variables are disabled.')

    args = parser.parse_args()

    required_args = [
        'project_id',
        'apphostingyaml_output_filepath',
        'dot_env_output_filepath',
        'backend_root_directory',
        'buildpack_config_output_filepath'
    ]

    for arg in required_args:
        if getattr(args, arg) is None:
            logging.error(f"--{arg} flag not specified.")
            raise ValueError(f"--{arg} is a required argument")

    async def run():
        try:
            # Initialize Secret Manager client
            with AsyncExitStack() as stack:
                secret_client = SecretManagerServiceClient()
                await stack.enter_async_context(secret_client)

                opts = {
                    'secret_client': secret_client,
                    'app_hosting_yaml_path': args.apphostingyaml_filepath,
                    'project_id': args.project_id,
                    'region': args.region,
                    'environment_name': args.environment_name,
                    'app_hosting_yaml_output_file_path': args.apphostingyaml_output_filepath,
                    'env_dereferenced_output_file_path': args.dot_env_output_filepath,
                    'backend_root_directory': args.backend_root_directory,
                    'buildpack_config_output_file_path': args.buildpack_config_output_filepath,
                    'firebase_config': args.firebase_config,
                    'firebase_webapp_config': args.firebase_webapp_config,
                    'server_side_env_vars': args.server_side_env_vars,
                    'app_hosting_preprocessed_path_for_pack': args.apphosting_preprocessed_path_for_pack
                }

                # Detect the apphosting.yaml path
                opts['app_hosting_yaml_path'] = await detect_app_hosting_yaml_path(args.workspace_path, args.backend_root_directory)

                await prepare(opts)
        except GoogleAPICallError as e:
            logging.error(f"Secret Manager API error: {e}")
            raise
        except Exception as e:
            logging.error(f"Unexpected error during preparation: {e}")
            raise

    try:
        asyncio.run(run())
    except Exception as e:
        logging.error(str(e))
        return 1

    return 0

if __name__ == "__main__":
    exit(main())
