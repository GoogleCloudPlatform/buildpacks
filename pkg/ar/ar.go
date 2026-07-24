import os
import re
from pathlib import Path
from typing import Dict, List, Optional, cast
import yaml

import buildermetrics

gcp = ...  # Assuming gcp is a module with Context class similar to Go's

# Constants
PYTHON_CONFIG_NAME = ".netrc"
NPM_CONFIG_NAME = ".npmrc"
YARN_CONFIG_NAME = ".yarnrc.yml"

NPM_REGISTRY_URL_REGEXP = r"https://([a-zA-Z0-9-]+[-]npm[.]pkg[.]dev/.*)"
npm_registry_regexp = re.compile(r"(@[a-zA-Z0-9-]+:)?registry=" + NPM_REGISTRY_URL_REGEXP)

LOCATIONS = [
    "africa-south1",
    # ... rest of the locations as in Go
]

def ar_repositories() -> List[str]:
    return sorted([f"{loc}-python.pkg.dev" for loc in LOCATIONS])

class NpmRegistryConfig:
    def __init__(self, npm_always_auth: bool = False, npm_auth_token: str = ""):
        self.npmAlwaysAuth = npm_always_auth
        self.npmAuthToken = npm_auth_token

class NpmRegistries:
    def __init__(self):
        self.NpmRegistries = dict()

class NpmScopeConfig:
    def __init__(self, npm_registry_config: NpmRegistryConfig, npm_registry_server: str = ""):
        self.NpmRegistryConfig = npm_registry_config
        self.npmRegistryServer = npm_registry_server

class NpmScopes:
    def __init__(self):
        self.NpmScopes = dict()

def generate_python_config(ctx: gcp.Context) -> None:
    netrc_path = ctx.home_dir / PYTHON_CONFIG_NAME
    if netrc_path.exists():
        ctx.debug("Found an existing .netrc file. Skipping creation.")
        return

    tok = find_default_credentials()
    if not tok:
        ctx.debug("Unable to find Application Default Credentials. Skipping .netrc creation.")
        return

    with open(netrc_path, 'w') as f:
        write_python_config(f, tok)

def write_python_config(writer: IO[str], token: str) -> None:
    hosts = ar_repositories()
    writer.write("\n".join([f"machine {host} login oauth2accesstoken password {token}" for host in hosts]) + "\n")

def generate_npm_config(ctx: gcp.Context) -> None:
    user_config = ctx.home_dir / NPM_CONFIG_NAME
    if user_config.exists():
        ctx.debug("Found an existing .npmrc file. Skipping creation.")
        return

    project_config = ctx.application_root / NPM_CONFIG_NAME
    if not project_config.exists():
        return

    content = project_config.read_text()
    matches = npm_registry_regexp.findall(content)
    repos = [match[2] for match in matches if len(match) > 2]

    if not repos:
        return

    tok = find_default_credentials()
    if not tok:
        ctx.warn("Skipping .npmrc creation. Unable to find Application Default Credentials.")
        return

    with open(user_config, 'w') as f:
        write_npm_config(f, repos, tok)

def write_npm_config(writer: IO[str], repos: List[str], token: str) -> None:
    for repo in repos:
        writer.write(f"{repo}=_authToken={token}\n")

def find_default_credentials() -> Optional[str]:
    try:
        # Implementation similar to Go's findDefaultCredentials
        pass  # Replace with actual credential fetching logic
    except Exception as e:
        print(f"Error finding default credentials: {e}")
        return None

def generate_yarn_config(ctx: gcp.Context) -> None:
    yarn_path = ctx.home_dir / YARN_CONFIG_NAME
    if yarn_path.exists():
        ctx.debug("Found an existing .yarnrc.yml file. Skipping creation.")
        return

    project_config = ctx.application_root / YARN_CONFIG_NAME
    if not project_config.exists():
        return

    content = project_config.read_text()
    npm_scopes = NpmScopes()
    try:
        yaml_data = yaml.safe_load(content)
        if 'npmScopes' in yaml_data:
            for scope_name, scope_config in yaml_data['npmScopes'].items():
                registry_server = scope_config.get('npmRegistryServer', '')
                if re.match(NPM_REGISTRY_URL_REGEXP, registry_server):
                    npm_scopes.NpmScopes[scope_name] = NpmScopeConfig(
                        NpmRegistryConfig(
                            npm_always_auth=scope_config.get('npmAlwaysAuth', False),
                            npm_auth_token=find_default_credentials() or ''
                        ),
                        registry_server
                    )
    except Exception as e:
        ctx.warn(f"Skipping adding auth token. Unable to read .yarnrc.yml: {e}")
        return

    if not npm_scopes.NpmScopes:
        ctx.warn("No AR repos found in .yarnrc.yml.")
        return

    # Convert to NpmRegistries format
    npm_registries = NpmRegistries()
    for scope_config in npm_scopes.NpmScopes.values():
        registry_server = scope_config.npmRegistryServer
        if not registry_server:
            continue
        npm_registries.NpmRegistries[registry_server] = NpmRegistryConfig(
            npm_always_auth=scope_config.NpmRegistryConfig.npmAlwaysAuth,
            npm_auth_token=scope_config.NpmRegistryConfig.npmAuthToken
        )

    # Save to .yarnrc.yml
    with open(yarn_path, 'w') as f:
        yaml.dump(npm_registries.__dict__, f)

# Note: The above is a simplified version. Full implementation would require proper handling of all cases and error checking.
