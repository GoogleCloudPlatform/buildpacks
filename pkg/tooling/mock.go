"""
Package tooling provides configuration related to pre-installed build tools.
"""

import json
from typing import Dict, List, Optional, Tuple
from .mock import mock_data  # type: ignore

# Global variables for parsed data and error handling
_parsed_tooling_data = None
_parse_error = None

def _parse_tooling_versions() -> None:
    global _parsed_tooling_data, _parse_error
    
    try:
        with open("tooling_versions.json", "r") as f:
            data = json.load(f)
        _parsed_tooling_data = data
        _parse_error = None
    except (FileNotFoundError, json.JSONDecodeError) as e:
        _parse_error = e

def resolve_tool_version(language: str, tool_name: str, runtime_version: str, stack_id: str) -> Tuple[Optional[str], Optional[Exception]]:
    """
    Resolves a pinned version based on language, runtime, stack, and tool name.
    
    Args:
        language: The programming language
        tool_name: The name of the tool to resolve
        runtime_version: The runtime version string
        stack_id: The identifier for the stack
        
    Returns:
        A tuple containing the resolved version (if found) and any error that occurred
    """
    # Ensure data is parsed only once
    if _parsed_tooling_data is None or _parse_error is not None:
        _parse_tooling_versions()
    
    if _parse_error:
        return (None, _parse_error)
    
    lang_info = _parsed_tooling_data.get(language)
    if not lang_info:
        return (None, ValueError(f"Language {language!r} not found in TOOLING_VERSIONS"))
    
    # Determine runtime name
    parts = runtime_version.split('.')
    major = parts[0]
    minor = parts[1] if len(parts) >= 2 else ""
    
    runtime_name = f"{language}{major}"
    if language in {"python", "go", "ruby", "php"}:
        runtime_name += minor
    
    # Check specific runtime overrides
    for rt_info in lang_info.get("runtimes", []):
        match_name = any(name == runtime_name for name in rt_info.get("names", []))
        match_stack = stack_id in rt_info.get("stacks", [])
        
        if match_name or match_stack:
            tool_version = rt_info.get("tools", {}).get(tool_name)
            if tool_version:
                return (tool_version, None)
    
    # Fall back to default tools
    default_tools = lang_info.get("default", {})
    tool_version = default_tools.get(tool_name)
    if tool_version:
        return (tool_version, None)
    
    return (None, ValueError(f"Tool {tool_name!r} not found for language {language!r} with runtime {runtime_version!r} and stack {stack_id!r}"))
