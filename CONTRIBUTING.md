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

The `gcpbuildpack` package implements general functionality that is shared
across buildpacks. In addition to the signature of the `build` and `detect`
functions, the package includes the `Context` struct that implements functions
to manipulate files and layers and to execute arbitrary commands.

Each buildpack has a single `main.go` file that implements a `detectFn` and
a `buildFn`:

`detectFn` is invoked through `/bin/detect`.
A buildpack signals that it can participate in the build unless it explicitly
opts out using `ctx.OptOut` or returns an error.

`buildFn` is invoked through `/bin/build`.

### Error attribution

The `gcpbuildpack` package supports error attribution to differentiate between
user and platform errors. Generally, any error that was triggered while
processing user code, such as installing dependencies or compiling a program,
should be attributed to the user. Errors that occur while manipulating files or
directories or when performing actions the user has no control over should be
attributed to the platform. Some errors may be ambiguous, such as downloading
dependencies from a remote repository, can be attributable to both the user
(wrong dependency version) and the platform (network error). In these cases,
errors should be attributed based on the **most likely** cause. It is much
more likely that the dependencies file has an incorrect version, undeclared
dependency, or an outdated lock file than it is for the network to be down.

Some examples:
* Reading package.json: PLATFORM (I/O error)
* Unmarshalling package.json: USER (invalid package.json file)
* Compiling a Go program: USER (syntax error, dependency not found, etc.)
* Downloading Go modules: USER (module not found more likely than network issue)

### Exec vs ExecUser

The `Context` struct provides convenience functions to execute arbitrary
commands. Use `Exec` and its derivatives for internal commands such as moving
files and `ExecUser` its derivatives for commands that depend on user input,
such as downloading dependencies or compiling a program. In all cases, prefer
using specialized functions when available on `Context` instead of `Exec`, for
example `ctx.Symlink` instead of `ln -s`.

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

