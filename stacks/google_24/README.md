# Buildpack Stack `google.24`

A minimal buildpack stack based on Ubuntu 24.04 LTS (Noble Numbat).

## Run Image

[us-docker.pkg.dev/serverless-runtimes/google-24/run](https://us-docker.pkg.dev/serverless-runtimes/google-24/run)

Installed Packages:

* `ca-certificates`
* `curl`
* `locales`
* `openssl`
* `tzdata`

## Build Image

[us-docker.pkg.dev/serverless-runtimes/google-24-full/build/go](https://us-docker.pkg.dev/serverless-runtimes/google-24-full/build/go)

## Building Images

The `run.Dockerfile` image require `CANDIDATE_NAME` as a `build-arg`. This is
unique identifier used to track releases, but any string can be provided for
local development.

To build the run image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag <REMOTE_RUN_URL>
```

## Run Tests

We use [container structure tests](https://github.com/GoogleContainerTools/container-structure-test)
to validate all stack images.

To test the run image:

```
container-structure-test test \
  --image <REMOTE_RUN_URL> \
  --config run_structure_test.yaml
```
