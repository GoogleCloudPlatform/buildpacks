import os
import io
import json
import tarfile
import logging
from typing import Optional, Any
from urllib.parse import urlparse
from pathlib import Path
import requests
import zstandard as zstd
import gzip
from google.cloud.artifactregistry_v1beta2 import ArtifactRegistryClient

logger = logging.getLogger(__name__)

class FetchError(Exception):
    pass

def tarball(url: str, dir_path: str, strip_components: int) -> None:
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    if "/zstd/" in url:
        with zstd.ZstdDecompressor().stream_reader(response.raw) as reader:
            _untar(reader, dir_path, strip_components)
    else:
        with gzip.open(response.raw, 'rb') as gzreader:
            _untar(gzreader, dir_path, strip_components)

def ar_versions(url: str, fallback_url: str, ctx: Any) -> list[str]:
    try:
        return ArtifactRegistryClient().list_tags(url)
    except Exception as e:
        logger.warning(f"Failed to list versions from {url}: {e}")
        logger.info(f"Falling back to {fallback_url}")
        return ArtifactRegistryClient().list_tags(fallback_url)

def ar_image(url: str, fallback_url: str, dir_path: str, strip_components: int) -> None:
    try:
        image = requests.get(url, stream=True)
        if image.status_code != 200:
            raise FetchError(f"Failed to download image from {url}")
        
        with tarfile.open(fileobj=image.raw, mode='r|*') as tar:
            _extract_tar(tar, dir_path, strip_components)
    except Exception as e:
        logger.warning(f"Failed to download image from {url}: {e}")
        ar_image(fallback_url, "", dir_path, strip_components)

def arg_generic_binary(binary_name: str, version: str, out_path: str) -> None:
    url = _build_arg_url(binary_name, version)
    
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    with open(out_path, 'wb') as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)
    os.chmod(out_path, 0o755)

def file_download(url: str, out_path: str) -> None:
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    with open(out_path, 'wb') as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)

def json_fetch(url: str) -> dict:
    response = requests.get(url)
    response.raise_for_status()
    return response.json()

def get_url(url: str, writer: io.TextIOBase) -> None:
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    for chunk in response.iter_content(chunk_size=8192):
        writer.write(chunk.decode('utf-8'))

def _untar(reader: io.RawIOBase, dir_path: str, strip_components: int) -> None:
    with tarfile.open(fileobj=reader, mode='r|*') as tar:
        _extract_tar(tar, dir_path, strip_components)

def _extract_tar(tar: tarfile.TarFile, dir_path: str, strip_components: int) -> None:
    for member in tar:
        if member.isdir():
            os.makedirs(os.path.join(dir_path, member.name), exist_ok=True)
        elif member.isfile():
            fpath = os.path.join(dir_path, member.name)
            os.makedirs(os.path.dirname(fpath), exist_ok=True)
            with open(fpath, 'wb') as f:
                f.write(tar.extractfile(member).read())

def _build_arg_url(binary_name: str, version: str) -> str:
    host = "artifactregistry.googleapis.com"
    project = "serverless-runtimes"
    
    if os.getenv('BUILD_ENV') == 'qual':
        project = "serverless-runtimes-qa"
        
    path = f"/download/v1/projects/{project}/locations/us-central1/repositories/universal-maker/files/x86-64:{version}:{binary_name}:download?alt=media"
    return f"https://{host}{path}"
