import sys
import os
import shutil
import importlib.util

if len(sys.argv) < 3:
    print("Internal error: Missing arguments.")
    sys.exit(1)

entrypoint = sys.argv[1]
cache_dir = sys.argv[2]

print("Starting bytecode cache generation...")

try:
    if os.path.exists(cache_dir):
        shutil.rmtree(cache_dir, ignore_errors=True)
    os.makedirs(cache_dir, exist_ok=False)

    spec = importlib.util.spec_from_file_location("entry_module", entrypoint)
    module = importlib.util.module_from_spec(spec)
    sys.modules["entry_module"] = module
    if spec.loader is not None:
        spec.loader.exec_module(module)

except Exception as e:
    print(f"Error during cache generation: {e}")
    sys.exit(1)
