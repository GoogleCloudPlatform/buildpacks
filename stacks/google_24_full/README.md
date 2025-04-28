# Buildpack Stack `google.gae.24`

A Ubuntu 24.04 LTS (Noble Numbat) buildpack stack used for Google App Engine
runtimes.

## Run Image

[us-docker.pkg.dev/serverless-runtimes/google-24-full/run](https://us-docker.pkg.dev/serverless-runtimes/google-24-full/run)

Available packages listed in [run-packages.txt](./run-packages.txt)

## Build Image

[us-docker.pkg.dev/serverless-runtimes/google-24-full/build/go](https://us-docker.pkg.dev/serverless-runtimes/google-24-full/build/go)

Available packages listed in [build-packages.txt](./build-packages.txt)

## Building Images

Both the `run.Dockerfile` and `build.Dockerfile` image require `CANDIDATE_NAME`
as a `build-arg`. This is unique identifier used to track releases, but any
string can be provided for local development.

To build the run image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag <REMOTE_RUN_URL>
```

To build the build image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file build.Dockerfile \
  --tag <REMOTE_BUILD_URL>
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

To test the build image:

```
container-structure-test test \
  --image <REMOTE_BUILD_URL> \
  --config build_structure_test.yaml
```