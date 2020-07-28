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

The GCP Buildpacks project is implemented in Go and uses
[Bazel](https://bazel.build/) as the build system. Throughout the document, all
commands are relative to project root and `bazel` is interchangable
with `blaze`.

### Installing project dependencies

Please follow the links below for installation instructions.

* [Bazel](https://docs.bazel.build/install.html)
* [Docker](https://store.docker.com/search?type=edition&offering=community)
* [pack](https://buildpacks.io/docs/install-pack/)
* [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test#installation)

Note: `docker`, `pack`, and `container-structure-test` must to be on `$PATH`
as interpreted by Bazel, which may differ from system `$PATH`. When using
Blaze, the programs or symlinks to them must be in `/usr/bin`.

The following command verifies that all dependencies are installed correctly
or prints further instructions:

```bash
bazel test --test_output=errors tools:check_dependencies_test
```

### Builder overview

Builder definitions and acceptance tests are located in the `builders`
directory.

* GCP
  * Builders used in [Cloud Code](https://cloud.google.com/code)/[Cloud Run](https://cloud.google.com/run)/[Skaffold](https://github.com/GoogleContainerTools/skaffold)
    and appropriate for the general use case, e.g. Kubernetes, local development.
* GAE
  * Builders used in [App Engine](https://cloud.google.com/appengine).
* GCF
  * Builders used in [Cloud Functions](https://cloud.google.com/functions).

### gcpbuildpack package

The `gcpbuildpack` package implements general functionality that is shared
across buildpacks. In addition to the signature of the `build` and `detect`
functions, the package includes the `Context` struct that implements functions
to manipulate files and layers and to execute arbitrary commands.

Each buildpack has a single `main.go` file that implements a `detectFn` and
a `buildFn`:

* `detectFn` is invoked through `/bin/detect`.
  A buildpack signals that it can participate in the build unless it explicitly
  opts out using `ctx.OptOut` or returns an error.

* `buildFn` is invoked through `/bin/build`.
  The responsibility of the build function is to create layers and populate them
  with data using a combination of Go and shell commands.

### Error attribution

The `gcpbuildpack` package supports error attribution to differentiate between
user and platform errors. Generally, any error that was triggered while
processing user code, such as installing dependencies or compiling a program,
should be attributed to the user. Errors that occur while manipulating files or
directories or when performing actions the user has no control over should be
attributed to the platform. Some errors may be ambiguous, such as downloading
dependencies from a remote repository, and attributable to both the user
(wrong dependency version) and the platform (network error). In these cases,
errors should be attributed based on the **most likely** cause. It is much
more likely that the dependencies file has an incorrect version, undeclared
dependency, or an outdated lock file than it is for the network to be down.

Some examples:
* Reading package.json: PLATFORM (I/O error)
* Unmarshalling package.json: USER (invalid package.json file)
* Compiling a Go program: USER (syntax error, dependency not found, etc.)
* Downloading Go modules: USER (module not found more likely than network issue)

### Using gcp.WithUserAttribution()

The `Context` struct provides convenience functions to execute arbitrary
commands. When calling `Exec`, use the option `gcp.WithUserAttribution`
for commands that depend on user input, such as downloading dependencies or
compiling a program. It is also possible to split failure attribution and
timing attribution, using `gcp.WithUserFailureAttribution`, or
`gcp.WithUserTimingAttribution`. In the absence of these options, attribution
is assigned to the system.
In all cases, prefer using specialized functions when available on `Context`
instead of `Exec`, for example `ctx.Symlink` instead of `ln -s`.

### Compiling a buildpack

```bash
export runtime=nodejs
export buildpack=npm
bazel build "cmd/${runtime}/${buildpack}:${buildpack}.tgz"
```

This will produce a tgz archive containing `buildpack.toml`, the `/bin/build`
and `/bin/detect` binaries, as well as any other files required by the
buildpack.

### Creating a builder

To create a builder for a given product and runtime, first pull or build the
stack images required by the builder:

```bash
# Optional, pull stack images required by the builder.
bazel run tools:pull_images gcp base
# Create a builder image tagged as <product>/<runtime>.
bazel build builders/gcp/base:builder.image
```

You can also rebuild the gcp/base stack images:

```bash
# Optional, build stack images required by the builder.
bazel run builders/gcp/base/stack:build
```

GAE and GCF stack images cannot be modified, to pull the latest stack images
and create a builder:

```bash
export product=gae
export runtime=nodejs12

# Optional, pull stack images required by the builder.
bazel run tools:pull_images "$product" "$runtime"
# Create a builder image tagged as gcp/base.
bazel build "builders/${product}/${runtime}:builder.image"
```

This will produce a builder image tagged as `<product>/<runtime>` in the local
Docker daemon.

### Updating Dependencies

If you would like to update any project dependencies, please file a new issue.

## Testing

Each builder has a set of acceptance tests that validate the builder by
building and running a set of applications. By default, the tests pull the
latest stack images from GCR. Running all acceptance tests is CPU, memory,
and network intensive, so we recommend only running tests for affected builders.

To run acceptance tests for the `gcp/base` builder, use the following command:

```bash
bazel test builders/gcp/base/acceptance/...
```

or more generally:

```bash
export product=gae
export runtime=nodejs12
bazel test "builders/${product}/${runtime}/acceptance/..."
```

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

## Common Problems

### Testing builder with `pack build` fails

Run `pack build ... -v` to produce more verbose debug output.

### `bazel test` fails on macOS or Windows

By default, `bazel` builds Go binaries for the current platform.  As GCP Buildpacks
are targeted to Linux-based container images, our `.bazelrc` configures builds for
the Linux AMD64 platform by default.

To run tests on the local platform, override the `--plaforms` as
follows:
```sh
bazel test --platforms="" pkg/...
```
