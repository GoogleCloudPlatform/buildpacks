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

import json
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional

@dataclass
class HeaderConfig:
    key: str
    value: str

@dataclass
class Header:
    source: str
    regex: Optional[str]
    headers: List[HeaderConfig]

@dataclass
class Run:
    service_id: str
    region: Optional[str]

@dataclass
class Rewrite:
    source: str
    regex: Optional[str]
    destination: Optional[str]
    function: Optional[str]
    run: Optional[Run]
    dynamic_links: bool

@dataclass
class Redirect:
    source: str
    regex: Optional[str]
    destination: str
    type_: int  # Using 'type_' to avoid keyword conflict

@dataclass
class HostingConfig:
    target: Optional[str]
    site: Optional[str]
    public: Optional[str]
    clean_urls: bool
    trailing_slash: Optional[bool]
    rewrites: List[Rewrite]
    redirects: List[Redirect]
    headers: List[Header]

def parse_firebase_config(path: str) -> List[HostingConfig]:
    firebase_json_path = Path(path)
    
    if not firebase_json_path.exists():
        return []
    
    try:
        with open(firebase_json_path, 'r') as f:
            data = json.load(f)
            
        hosting_configs = []
        
        if isinstance(data.get('hosting'), list):
            for config in data['hosting']:
                hosting_config = HostingConfig(
                    target=config.get('target'),
                    site=config.get('site'),
                    public=config.get('public'),
                    clean_urls=config.get('cleanUrls', False),
                    trailing_slash=config.get('trailingSlash'),
                    rewrites=_parse_rewrites(config.get('rewrites', [])),
                    redirects=_parse_redirects(config.get('redirects', [])),
                    headers=_parse_headers(config.get('headers', []))
                )
                hosting_configs.append(hosting_config)
        elif isinstance(data.get('hosting'), dict):
            config = data['hosting']
            hosting_config = HostingConfig(
                target=config.get('target'),
                site=config.get('site'),
                public=config.get('public'),
                clean_urls=config.get('cleanUrls', False),
                trailing_slash=config.get('trailingSlash'),
                rewrites=_parse_rewrites(config.get('rewrites', [])),
                redirects=_parse_redirects(config.get('redirects', [])),
                headers=_parse_headers(config.get('headers', []))
            )
            hosting_configs.append(hosting_config)
            
        return hosting_configs
    except json.JSONDecodeError as e:
        raise ValueError(f"Failed to parse firebase.json: {e}") from e

def _parse_rewrites(rewrites_json):
    rewrites = []
    for rw in rewrites_json:
        rewrite = Rewrite(
            source=rw.get('source'),
            regex=rw.get('regex'),
            destination=rw.get('destination'),
            function=rw.get('function'),
            run=_parse_run(rw.get('run')),
            dynamic_links=rw.get('dynamicLinks', False)
        )
        rewrites.append(rewrite)
    return rewrites

def _parse_run(run_json):
    if run_json:
        return Run(
            service_id=run_json.get('serviceId'),
            region=run_json.get('region')
        )
    return None

def _parse_redirects(redirects_json):
    redirects = []
    for red in redirects_json:
        redirect = Redirect(
            source=red.get('source'),
            regex=red.get('regex'),
            destination=red.get('destination'),
            type_=red.get('type')
        )
        redirects.append(redirect)
    return redirects

def _parse_headers(headers_json):
    headers = []
    for hdr in headers_json:
        header = Header(
            source=hdr.get('source'),
            regex=hdr.get('regex'),
            headers=_parse_header_configs(hdr.get('headers', []))
        )
        headers.append(header)
    return headers

def _parse_header_configs(headers_config_json):
    configs = []
    for config in headers_config_json:
        configs.append(HeaderConfig(
            key=config['key'],
            value=config['value']
        ))
    return configs
