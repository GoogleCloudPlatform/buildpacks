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

[gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/build](https://gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/build)

Installed Packages:

* `build-essential`
* `ca-certificates`
* `curl`
* `git`
* `locales`
* `openssl`
* `tar`
* `tzdata`
* `unzip`
* `xz-utils`
* `zip`

## Building Images

Both the `run.Dockerfile` and `build.Dockerfile` image require `CANDIDATE_NAME`
as a `build-arg`. This is unique identifier used to track releases, but any
string can be provided for local development.

To build the run image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/run
```

To build the build image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file build.Dockerfile \
  --tag gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/build
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

To test the build image:

```
container-structure-test test \
  --image gcr.io/gae-runtimes/buildpacks/stacks/google-min-22/build \
  --config build_structure_test.yaml
```