import argparse
import json
import os
import re
from pathlib import Path

def parse_go_file(file_path):
    with open(file_path, 'r') as f:
        lines = f.readlines()
    
    package_name = None
    imports = set()
    in_import_block = False
    current_imports = []
    
    for line in lines:
        stripped_line = line.strip()
        
        # Check if it's a package statement
        if not in_import_block and re.match(r'^package\s+\w+', stripped_line):
            match = re.match(r'package (\w+)', stripped_line)
            if match:
                pkg_name = match.group(1)
                if package_name is None:
                    package_name = pkg_name
                elif package_name != pkg_name:
                    raise ValueError(f"Package name mismatch: {pkg_name} vs {package_name}")
        
        # Check for import statements
        if stripped_line.startswith('import'):
            if '(' in stripped_line:
                # Multi-line import block starting with (
                in_import_block = True
                parts = re.findall(r'"[^"]*"', stripped_line)
                for part in parts:
                    current_imports.append(part.strip('"'))
            else:
                # Single-line import like import "path"
                parts = re.findall(r'"[^"]*"', stripped_line)
                for part in parts:
                    imports.add(part.strip('"'))
        
        elif in_import_block:
            if stripped_line.endswith(')'):
                # End of import block
                in_import_block = False
                for imp in current_imports:
                    imports.add(imp)
                current_imports.clear()
            else:
                # Inside import block, collect all quoted strings
                parts = re.findall(r'"[^"]*"', line)
                for part in parts:
                    current_imports.append(part.strip('"'))
    
    return package_name, imports

def main():
    parser = argparse.ArgumentParser(description='Extract Go package and imports from a directory.')
    parser.add_argument('--dir', type=str, required=True,
                       help='Directory containing .go files')
    args = parser.parse_args()
    
    dir_path = Path(args.dir)
    if not dir_path.is_dir():
        raise FileNotFoundError(f"Directory {args.dir} does not exist.")
    
    package_name = None
    all_imports = set()
    
    for go_file in dir_path.glob("*.go"):
        try:
            pkg, imp = parse_go_file(go_file)
            if package_name is None:
                package_name = pkg
            elif pkg != package_name:
                raise ValueError(f"Package name mismatch: {pkg} vs {package_name}")
            
            all_imports.update(imp)
        except Exception as e:
            print(f"Error parsing file {go_file}: {e}")
            exit(1)
    
    if package_name is None:
        raise ValueError("No package found in any .go files.")
    
    result = {
        'name': package_name,
        'imports': list(all_imports)
    }
    
    print(json.dumps(result, indent=2))

if __name__ == '__main__':
    main()
