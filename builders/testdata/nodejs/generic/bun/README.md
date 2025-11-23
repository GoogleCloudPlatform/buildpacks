# Bun Test Application

This is a simple Express application used to test the Bun buildpack.

## Setup

To generate the `bun.lockb` file, you need to have Bun installed:

```bash
# Install Bun
curl -fsSL https://bun.sh/install | bash

# Generate lockfile
cd builders/testdata/nodejs/generic/bun
bun install
```

This will create a `bun.lockb` file that pins the exact version of Express (4.18.2) for testing.

## What this tests

- The buildpack detects the presence of `bun.lockb` and `package.json`
- Bun is installed correctly
- Dependencies are installed using Bun
- The application runs correctly with the installed dependencies
- The Express version matches what's in the lockfile
