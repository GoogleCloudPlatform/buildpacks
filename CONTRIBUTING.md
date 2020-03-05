# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution;
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## Community Guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Building Buildpacks

The GCP Buildpacks project uses Bazel for building.  Implementation is in Go.

* [Install Bazel](https://docs.bazel.build/versions/master/install.html)

### Builders

Builder definitions and acceptance tests are located in the `builders`
directory.

* GCP
  * Builders used in Cloud Code/Cloud Run/Skaffold and appropriate for the
    general use case, e.g. Kubernetes, local development.
* GAE
  * Builders used for runtimes in App Engine.
* GCF
  * Builders used for runtimes in Cloud Functions.


### gcpbuildpack package

TODO: How layers/caching works in our buildpacks.

### Error attribution

The `gcpbuildpack` package supports error attribution to differentiate between
user and operator errors.

TODO: Exaplain which calls should be user- and operator-attributed.

### Compiling a buildpack

```bash
export runtime=nodejs
export buildpack=npm
bazel build "cmd/${runtime}/${buildpack}:${buildpack}.tar"
```

This will produce a tar archive containing the `/bin/build` and `/bin/detect`
binaries, as well as any other files required by the buildpack.

### Creating a builder

To create a builder for a given product and runtime, first pull or build the
stack images required by the builder:

```bash
# Optional, pull stack images required by the builder.
./tools/pull-images.sh gcp base
# Create a builder image tagged as gcp/base.
bazel build builders/gcp/base:builder.image
```

or more generally:

```bash
export product=gae
export runtime=nodejs12

# Optional, pull stack images required by the builder.
./tools/pull-images.sh "$product" "$runtime"
# Create a builder image tagged as gcp/base.
bazel build "builders/${product}/${runtime}:builder.image"
```

This will produce a builder in the form of a Docker image tagged as
`<product>/<runtime>`.

### Updating Dependencies

[Gazelle](https://github.com/bazelbuild/bazel-gazelle) is a buildfile generator
for bazel, and synchronizes the WORKSPACE's `go_repository` specs with the versions
specified in `go.mod`.  After updating the `go.mod` then run:
```sh
bazel run //:gazelle -- update-repos -from_file=go.mod
```

## Testing

To run acceptance tests, perform the following:
```bash
bazel test builders/gcp/base:all

### Cleaning up Docker artifacts

The acceptance tests attempt to clean up containers and images after they
finish running, but there maybe some left-over data that can end up taking
a significant amount of storage space.

```bash
# Remove all stopped containers.
docker rm -f $(docker ps -a -q)
# Remove all untagged images.
docker rmi $(docker images --filter "dangling=true" -q --no-trunc)
# Remove pack cache volumes.
docker volume prune
# Delete images older than 30 days.
docker image prune --all --filter until=720h
```

