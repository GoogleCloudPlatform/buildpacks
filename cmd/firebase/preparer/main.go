# Copyright 2024 Google LLC
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
from typing import Dict, Any

import google.auth
from google.cloud import secretmanager
from firebase.util.filesystem import detect_app_hosting_yaml_path  # type: ignore
from firebase.preparer import prepare  # type: ignore
from gcpbuildpack.context import GCPBuildContext  # type: ignore
from faherror import FahError, InternalError  # type: ignore

def main() -> None:
    parser = argparse.ArgumentParser(description='Run preprocessing steps for App Hosting backend builds.')
    parser.add_argument('--apphostingyaml_filepath', default='', help='File path to user defined apphosting.yaml')
    parser.add_argument('--workspace_path', default='/workspace', help='File path to the workspace directory')
    parser.add_argument('--project_id', required=True, help='User\'s GCP project ID')
    parser.add_argument('--region', default='', help='Current GCP Region. Used to expand resource IDs.')
    parser.add_argument('--environment_name', default='', help='Environment name tied to the build, if applicable')
    parser.add_argument('--apphostingyaml_output_filepath', required=True, help='File path to write the validated and formatted apphosting.yaml to')
    parser.add_argument('--apphosting_preprocessed_path_for_pack', default='/workspace/apphosting_preprocessed',
                       help='File path to write the preprocessed apphosting.yaml to for pack step to consume')
    parser.add_argument('--dot_env_output_filepath', required=True, help='File path to write the output .env file to')
    parser.add_argument('--backend_root_directory', required=True, help='File path to the application directory specified by the user')
    parser.add_argument('--buildpack_config_output_filepath', required=True, help='File path to write the buildpack config to')
    parser.add_argument('--firebase_config', default='', help='JSON serialized Firebase config used by Firebase Admin SDK')
    parser.add_argument('--firebase_webapp_config', default='', help='JSON serialized Firebase config used by Firebase Client SDK')
    parser.add_argument('--server_side_env_vars', default='',
                       help='List of server side env vars to set. An empty string indicates server side environment variables are disabled.')

    args = parser.parse_args()

    if not args.project_id:
        raise ValueError("--project_id flag not specified.")
    
    if not args.apphostingyaml_output_filepath:
        raise ValueError("--apphostingyaml_output_filepath flag not specified.")
    
    if not args.dot_env_output_filepath:
        raise ValueError("--dot_env_output_filepath flag not specified.")
    
    if not args.backend_root_directory:
        raise ValueError("--backend_root_directory flag not specified.")
    
    if not args.buildpack_config_output_filepath:
        raise ValueError("--buildpack_config_output_filepath flag not specified.")

    # Initialize SecretManager client
    credentials, project = google.auth.default()
    secret_client = secretmanager.SecretManagerServiceClient(credentials=credentials)

    opts: Dict[str, Any] = {
        'secret_client': secret_client,
        'app_hosting_yaml_path': args.apphostingyaml_filepath,
        'project_id': args.project_id,
        'region': args.region,
        'environment_name': args.environment_name,
        'app_hosting_output_file_path': args.apphostingyaml_output_filepath,
        'env_dereferenced_output_file_path': args.dot_env_output_filepath,
        'backend_root_directory': args.backend_root_directory,
        'buildpack_config_output_file_path': args.buildpack_config_output_filepath,
        'firebase_config': args.firebase_config,
        'firebase_webapp_config': args.firebase_webapp_config,
        'server_side_env_vars': args.server_side_env_vars,
        'apphosting_preprocessed_path_for_pack': args.apphosting_preprocessed_path_for_pack
    }

    # Detect app_hosting.yaml path
    opts['app_hosting_yaml_path'] = detect_app_hosting_yaml_path(args.workspace_path, args.backend_root_directory)

    gcp_ctx = GCPBuildContext()

    try:
        prepare(opts)
    except FahError as fe:
        raise ValueError(fe) from fe
    except Exception as e:
        raise InternalError(e) from e

if __name__ == "__main__":
    main()
