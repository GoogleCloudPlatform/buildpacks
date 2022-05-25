# Buildpack Stack `google.gae.22`

A Ubuntu 22.04 LTS (Jammy Jellyfish) buildpack stack used for Google App Engine
runtimes.

## Run Image

[gcr.io/buildpacks/stacks/google-gae-22/run](https://gcr.io/buildpacks/stacks/google-gae-22/run)

Available packages listed in [run-packages.txt](./run-packages.txt)

## Build Image

[gcr.io/buildpacks/stacks/google-gae-22/build](https://gcr.io/buildpacks/stacks/google-gae-22/build)

Available packages listed in [build-packages.txt](./build-packages.txt)

## Building Images

Both the `run.Dockerfile` and `build.Dockerfile` image require `CANDIDATE_NAME`
as a `build-arg`. This is unique identifier used to track releases, but any
string can be provided for local development.

To build the run image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag gcr.io/buildpacks/stacks/google-gae-22/run
```

To build the build image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file build.Dockerfile \
  --tag gcr.io/buildpacks/stacks/google-gae-22/build
```

## Run Tests

We use [container structure tests](https://github.com/GoogleContainerTools/container-structure-test)
to validate all stack images.

To test the run image:

```
container-structure-test test \
  --image gcr.io/buildpacks/stacks/google-gae-22/run \
  --config run_structure_test.yaml
```

To test the build image:

```
container-structure-test test \
  --image gcr.io/buildpacks/stacks/google-gae-22/build \
  --config build_structure_test.yaml
```