# Zstandard (zstd) Docker Image Compression Guide

This guide covers how to transcode an existing Docker image's layers into native
OCI `zstd` compression, force BuildKit to respect different compression levels,
and mathematically verify the network size reduction directly from the registry
manifest.

## 1. Environment Setup

By default, the standard local Docker daemon ignores advanced compression output
flags. To bypass this, you must create an isolated BuildKit container to handle
the transcoding.

```bash
# 1. Create a 1-line Dockerfile pointing to your target image
echo "FROM us-east1-docker.pkg.dev/serverless-runtimes/runtimes-ubuntu2204/nodejs:22.22.0" > Dockerfile.zstd

# 2. Spin up a dedicated advanced builder (Run this once per machine/environment)
docker buildx create --name zstd-builder --driver docker-container --use

```

## 2. The Baseline Compression (Level 3)

Level 3 is the Zstandard default. It provides an optimal balance: significantly
outperforming standard `gzip` in file size, while keeping build times fast and
decompression speeds nearly instantaneous.

```bash
docker buildx build \
  -f Dockerfile.zstd \
  --output type=registry,name=us-east1-docker.pkg.dev/<PROJECT_ID>/utils/nodejs:22.22.0-zstd-l3,compression=zstd,compression-level=3,force-compression=true,oci-mediatypes=true \
  .

```

## 3. Forcing Higher Compression (Levels 8 - 22)

BuildKit aggressively caches exported layers. If you run the build command again
with a higher compression level on the exact same source image, BuildKit will
silently reuse the file it already generated. You must manually clear the cache
first.

```bash
# 1. Wipe the BuildKit internal export cache
docker buildx prune --builder zstd-builder --all --force

# 2. Run the build with the new level (e.g., Level 8)
docker buildx build \
  --no-cache \
  -f Dockerfile.zstd \
  --output type=registry,name=us-east1-docker.pkg.dev/<PROJECT_ID>/utils/nodejs:22.22.0-zstd-l8,compression=zstd,compression-level=8,force-compression=true,oci-mediatypes=true \
  .

```

> **Note:** Due to internal Go library limitations inside Docker BuildKit
> (`klauspost/compress/zstd`), requesting levels `9` through `22` will
> automatically cap out at roughly Level `11` logic (`SpeedBest`).

## 4. Verifying Compression Format and Size

You cannot use standard `docker inspect` to check layer compression formats, as
it only measures the uncompressed data residing on your local disk. You must
inspect the raw OCI manifest sitting remotely in Artifact Registry.

### Step A: Get the Architecture Digest

Because we used `oci-mediatypes=true`, the registry holds an OCI Index. Run this
command against your new tag to get the index list:

```bash
docker buildx imagetools inspect --raw us-east1-docker.pkg.dev/<PROJECT_ID>/utils/nodejs:22.22.0-zstd-l3

```

Look at the JSON output and copy the `digest` hash specifically for the `amd64`
linux platform (e.g., `sha256:a03539...`).

### Step B: Inspect the Specific Manifest

Append `@` and the hash you just copied to the end of your image name:

```bash
docker buildx imagetools inspect --raw us-east1-docker.pkg.dev/<PROJECT_ID>/utils/nodejs:22.22.0-zstd-l3@sha256:a03539...

```

### Step C: Read the Output

Inside the `"layers"` array of the JSON output, verify the two critical metrics:

1.  **Format:** `"mediaType": "application/vnd.oci.image.layer.v1.tar+zstd"`
    (Proves the layer is natively compressed using Zstandard).
2.  **Size:** `"size": 46794081` (The exact compressed network payload size in
    bytes).

```
```
