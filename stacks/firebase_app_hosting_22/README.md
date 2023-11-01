# Buildpack Stack `firebase.apphosting.22`

A buildpack stack for Firebase App Hosting based on `google.22`.

## Run Image

[gcr.io/buildpacks/firebase-app-hosting-22/run](https://gcr.io/buildpacks/firebase-app-hosting-22/run)

Installed packages include all of the packages from [gcr.io/buildpacks/google-22/run](https://gcr.io/buildpacks/google-22/run).

## Build Image

[gcr.io/buildpacks/firebase-app-hosting-22/build](https://gcr.io/buildpacks/firebase-app-hosting-22/build)

Installed packages include all of the packages from [gcr.io/buildpacks/google-22/build](https://gcr.io/buildpacks/google-22/build).

## Building Images

Both the `run.Dockerfile` and `build.Dockerfile` image require `CANDIDATE_NAME`
as a `build-arg`. This is unique identifier used to track releases, but any
string can be provided for local development.

To build the run image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file run.Dockerfile \
  --tag gcr.io/buildpacks/firebase-app-hosting-22/run
```

To build the build image:

```
docker build . \
  --build-arg CANDIDATE_NAME=test \
  --file build.Dockerfile \
  --tag gcr.io/buildpacks/firebase-app-hosting-22/build
```

## Run Tests

We use [container structure tests](https://github.com/GoogleContainerTools/container-structure-test)
to validate all stack images.

To test the run image:

```
container-structure-test test \
  --image gcr.io/buildpacks/firebase-app-hosting-22/run \
  --config run_structure_test.yaml
```


To test the build image:

```
container-structure-test test \
  --image gcr.io/buildpacks/firebase-app-hosting-22/build \
  --config build_structure_test.yaml
```