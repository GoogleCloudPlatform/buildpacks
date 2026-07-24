# Copyright 2022 Google LLC
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

"""
Package version provides utility methods for working with semantic versions.
"""

from dataclasses import dataclass
import re
from typing import List, Optional

import semver


skip_keywords = ["deprecated", "public-image", "latest"]


@dataclass
class ResolveParams:
    no_sanitize: bool = False


def without_sanitization() -> dict:
    """Indicates the return value should not have any prefix trimmed or 0s appended."""
    return {"no_sanitize": True}


def resolve_version(
    constraint: str,
    versions: List[str],
    **kwargs
) -> str:
    """
    Finds the largest version in a list of semantic versions that satisfies the provided constraint.
    If no version in the list satisfies the constraint, it returns an error.
    
    Args:
        constraint: The semver constraint to match against versions.
        versions: A list of version strings to evaluate.
        kwargs: Additional options for resolution (e.g., 'no_sanitize').
        
    Returns:
        The highest version that matches the constraint.
        
    Raises:
        ValueError: If no matching version is found or if there's an error parsing versions.
    """
    params = ResolveParams(**kwargs)
    
    if not constraint:
        constraint = "*"
    
    try:
        c = semver.Constraints(constraint)
    except ValueError as e:
        raise ValueError(f"Invalid constraint '{constraint}': {e}") from e
    
    valid_versions = []
    for version in versions:
        if should_skip_version(version, skip_keywords):
            continue
        try:
            s = semver.Version.parse(version)
            valid_versions.append(s)
        except ValueError as e:
            raise ValueError(f"Failed to parse version '{version}': {e}") from e
    
    # Sort versions in descending order
    sorted_versions = sorted(valid_versions, reverse=True, key=lambda x: (x.major, x.minor, x.patch))
    
    for s in sorted_versions:
        if c.match(s):
            if params.no_sanitize:
                return str(s)
            return f"{s.major}.{s.minor}.{s.patch}"
    
    raise ValueError(f"Failed to resolve version matching constraint '{constraint}' against {valid_versions}")


def should_skip_version(version: str, keywords: List[str]) -> bool:
    """
    Determines if a version should be skipped based on specific criteria.
    
    Args:
        version: The version string to evaluate.
        keywords: Keywords that indicate the version should be skipped.
        
    Returns:
        True if the version should be skipped, False otherwise.
    """
    if is_release_candidate(version):
        return True
    for keyword in keywords:
        if version.lower().startswith(keyword.lower()):
            return True
    return False


def is_exact_semver(constraint: str) -> bool:
    """
    Checks if a given string is an exact semantic version.
    
    Args:
        constraint: The string to check.
        
    Returns:
        True if the string is an exact semver, False otherwise.
    """
    parts = constraint.split('.')
    return len(parts) == 3 and all(part.isdigit() for part in parts[:3])


def is_release_candidate(constraint: str) -> bool:
    """
    Checks if a given string is a release candidate version.
    
    Args:
        constraint: The string to check.
        
    Returns:
        True if the string is a release candidate, False otherwise.
    """
    pattern = r'^\d+\.\d+\.\d+(rc|RC|beta)\d+$'
    return re.match(pattern, constraint) is not None
