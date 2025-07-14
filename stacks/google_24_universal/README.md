# Buildpack Stack `google.24`

A buildpack stack based on Ubuntu 24.04 LTS (Noble Numbat).

## Run Image

[us-docker.pkg.dev/serverless-runtimes/google-24-full/run/universal](https://us-docker.pkg.dev/serverless-runtimes/google-24-full/run/universal)
or
[gcr.io/buildpacks/google-24/run](https://gcr.io/buildpacks/google-24/run)

Available packages listed in [run-packages.txt](./run-packages.txt).

## Build Image
[us-docker.pkg.dev/serverless-runtimes/google-24-full/build/universal](https://us-docker.pkg.dev/serverless-runtimes/google-24-full/build/universal)
or
[gcr.io/buildpacks/google-24/build](https://gcr.io/buildpacks/google-24/build)

Available packages listed in [build-packages.txt](./build-packages.txt).

## Building Images

Both the `run.Dockerfile` and `build.Dockerfile` image require `CANDIDATE_NAME`
as a `build-arg`. This is unique identifier used to track releases, but any
string can be provided for local development.

To build the run image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag gcr.io/buildpacks/google-24/run
```

To build the build image:

```
DOCKER_BUILDKIT=1 docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file build.Dockerfile \
  --tag gcr.io/buildpacks/google-24/build
```

## Run Tests

We use [container structure tests](https://github.com/GoogleContainerTools/container-structure-test)
to validate all stack images.

To test the run image:

```
container-structure-test test \
  --image gcr.io/buildpacks/google-24/run \
  --config run_structure_test.yaml
```

To test the build image:

```
container-structure-test test \
  --image gcr.io/buildpacks/google-24/build \
  --config build_structure_test.yaml
```