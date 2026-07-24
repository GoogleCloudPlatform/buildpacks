# SAMPLE-PIP-DEPENDENCY

This is a simple package to test pip vendored deps.

# Generate distribution package
Use "python3 -m build" in python-package to generate .whl binary. pyproject.toml has all details to create distribution package.
Leave tests/ and __init__.py empty - these are merely required to generate distribtuion package.

# Usage
Import this as "from sample_pip_dependency import sample" and use as sample.helloworld() to test.
