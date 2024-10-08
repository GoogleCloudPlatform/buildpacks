# Buildpack Stack `google.min.22`

A minimal buildpack stack based on Ubuntu 22.04 LTS (Jammy Jellyfish).

## Run Image

[gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/run](https://gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/run)

Installed Packages:

* `ca-certificates`
* `locales`
* `openssl`
* `tzdata`

## Build Image

[gcr.io/gae-runtimes/buildpacks/stacks/google-gae-22/build](https://gcr.io/gae-runtimes/buildpacks/stacks/google-gae-22/build)

## Building Images

The `run.Dockerfile` image require `CANDIDATE_NAME` as a `build-arg`. This is
unique identifier used to track releases, but any string can be provided for
local development.

To build the run image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/run
```

## Run Tests

We use [container structure tests](https://github.com/GoogleContainerTools/container-structure-test)
to validate all stack images.

To test the run image:

```
container-structure-test test \
  --image gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/run \
  --config run_structure_test.yaml
```
