# Google Cloud Buildpacks

This repository contains a set of builders and buildpacks designed to run on
Google Cloud's container platforms:
 [Cloud Run](https://cloud.google.com/run),
 [GKE](https://cloud.google.com/kubernetes-engine),
 [Anthos](https://cloud.google.com/anthos),
 and [Compute Engine runing Container-Optimized OS](https://cloud.google.com/container-optimized-os/docs).
 They are also used as the build system for [App Engine](https://cloud.google.com/appengine) and [Cloud Functions](https://cloud.google.com/functions).
 They are 100% compatible with [CNCF Buildpacks](https://buildpacks.io/).

## Quickstart

1. [Install Docker](https://store.docker.com/search?type=edition&offering=community)
1. [Install the pack tool (a CLI for running Buildpacks)](https://buildpacks.io/docs/install-pack/)
1. Clone the [sample apps](https://github.com/GoogleCloudPlatform/buildpack-samples):
    ```
    git clone https://github.com/GoogleCloudPlatform/buildpack-samples.git
    cd buildpack-samples
    ```
1. Pick a sample and build it, for instance with `sample-go`:
    ```
    cd sample-go
    pack build --builder gcr.io/buildpacks/builder sample-go
    ```
1. Run it with docker, like:
    ```
    docker run --rm -p 8080:8080 sample-go
    ```
See the [Usage section](#usage) for more details.


## Concepts

To read more, see Buildpack project
[documentation](https://buildpacks.io/docs/concepts/).

  * **[Builder](https://buildpacks.io/docs/concepts/components/builder/)** A container image that contains buildpacks and detection order in which builds are executed.
* **[Buildpack](https://buildpacks.io/docs/concepts/components/buildpack/)** An executable that "inspects your app source code and formulates a plan to build and run your application".
* **Buildpack Group** Several buildpacks which together provide support for a
specific language or framework.
* **[Run Image](https://buildpacks.io/docs/concepts/components/stack/)** The container image that serves as the base for the built application.


## Generic Builder and Buildpacks

This is a general purpose builder that creates container images designed to run on most
platforms (e.g. Kubernetes / Anthos, Knative / Cloud Run, Container OS, etc),
and should be used by the majority of users. The builder attempts to autodetect
the language of your source code, and can also build functions compatible with
the [Google Cloud Function Framework](https://cloud.google.com/functions/docs/functions-framework) by [setting the GOOGLE_FUNCTION_TARGET env var](#configuration).

The generic builder is hosted at `gcr.io/buildpacks/builder`.

Supported languages include:


|Runtime       |App Support | Function Support  |
|--------------|:----------:|:-----------------:|
| Go 1.10 +    | ✓          | ✓                 |
| Node.js 10 + | ✓          | ✓                 |
| Python 3.7 + | ✓          | ✓                 |
| Java 8, 11   | ✓          |                   |
| .Net 3 +     | ✓          |                   |

## App Engine and Cloud Function Builders and Buildpacks

These builders create container images designed to run on Google Cloud's App
Engine and Functions services. Most of the buildpacks are
identical to those in the generic builder.

Compared to the generic builder, there are two primary differences. First,
there are additional buildpacks which add transformations specific to each
service. Second, in order to optimize execution speed, each
language has a seperate builder.

## Usage

The Google Cloud Buildpacks project provides builder images suitable for use
with
[pack](https://github.com/buildpacks/pack),
[kpack](https://github.com/pivotal/kpack),
[tekton](https://github.com/tektoncd/catalog/tree/master/buildpacks),
[skaffold](https://github.com/GoogleContainerTools/skaffold/tree/master/examples/buildpacks),
and other tools that support the Buildpacks v3 specification.


### Building an application

The following command invokes [pack](https://github.com/buildpacks/pack) to
apply the general builder to build the application in the current directory, and then containerizes the result into a local container image named `<app-name>`.

```bash
pack build <app-name> --builder gcr.io/buildpacks/builder
```

The application you built can then be executed locally:

```bash
docker run --rm -p 8080:8080 <app-name>
```

You can set Cloud Buildpacks as your default:

```bash
pack set-default-builder gcr.io/buildpacks/builder
```

And you can publish the built image to the cloud directly with [pack](https://github.com/buildpacks/pack):

```bash
pack build --publish gcr.io/YOUR_PROJECT_ID/APP_NAME
```


### Extending the run image

If your application requires additional system packages to be installed and
available when it runs, you can accomplish this by customizing the **run**
container image.

```bash
cat > run.Dockerfile << EOF
FROM gcr.io/buildpacks/gcp/run
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
  imagemagick && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*
USER cnb
EOF

docker build -t my-run-image -f run.Dockerfile .
```

To use the custom run image with pack:

```bash
pack build my-app --builder gcr.io/buildpacks/builder --run-image my-run-image
```
### Extending the builder image

If you require certain packages for **building** your application, create a custom
builder image based on the base builder:

```bash
cat > builder.Dockerfile << EOF
FROM gcr.io/buildpacks/builder
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
  subversion && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*
USER cnb
EOF

docker build -t my-builder-image -f builder.Dockerfile .
```

To use the custom builder with pack:

```bash
pack build my-app --builder my-builder-image
```

### Configuration

Google Cloud Buildpacks support configuration using a set of **environment
variables** that are supported across runtimes.

* `GOOGLE_ENTRYPOINT`
  * Specifies the command which is run when the container is executed; equivalent to [entrypoint](https://docs.docker.com/engine/reference/builder/#entrypoint) in a Dockerfile.
  * **Example:** `gunicorn -p :8080 main:app` for Python. `java -jar target/myjar.jar` for Java.
* `GOOGLE_FUNCTION_TARGET`
  * For use with source code built around the [Google Cloud Functions Framework](https://cloud.google.com/functions/docs/functions-framework). Specifies the name of the function to be invoked.
  * **Example:** `myFunction` will cause the Functions Framework to invoke the function of the same name.
* `GOOGLE_RUNTIME`
  * If specified, forces the runtime to opt-in. If the runtime buildpack appears in multiple groups, the first group will be chosen, consistent with the buildpack specification. *(only works with buildpacks which install language runtimes)*.
  * **Example:** `nodejs` will cause the nodejs/runtime buildpack to opt-in.
* `GOOGLE_RUNTIME_VERSION`
  * If specified, overrides the runtime version to install.
*(only works with buildpacks which install language runtimes)*
  * **Example:** `13.7.0` for Node.js, `1.14.1` for Go. `11.0.6+10` for Java.
* `GOOGLE_BUILDABLE`
  * *(only applicable to compiled languages)* Specifies path to a buildable unit.
  * **Example:** `./maindir` for Go will build the package rooted at maindir.
* `GOOGLE_DEVMODE`
  * Enables the development mode buildpacks. This is used by [Skaffold](https://skaffold.dev) to enable live local development where changes to your source code trigger automatic container rebuilds. To use, install Skaffold and run `skaffold dev`.
  * **Example:** `true`, `True`, `1` will enable development mode.
* `GOOGLE_CLEAR_SOURCE`
  * *(only applicable to Go)* Clears source after the application is built. If the application depends on static files, such as Go templates, setting this variable may cause the application to misbehave.
  * **Example:** `true`, `True`, `1` will clear the source.

Certain language buildpacks support other environment variables.

### Go Buildpacks

* `GOOGLE_GOGCFLAGS`
  * Passed to `go build` and `go run` as `-gcflags value` with no interpretation.
  * **Example:** `all=-N -l` enables race condition analysis and changes how source filepaths are recorded in the binary.
* `GOOGLE_GOLDFLAGS`
  * Passed to `go build` and `go run` as `-ldflags value` with no interpretation.
  * **Example:** `-s -w` is used to strip and reduce binary size.

## Known Limitations

* **General**:
  * Caching is project-specific, not cross-project. Dependencies, such as the JDK, cannot be shared across projects and need to be redownloaded on first build.
* **Java**:
  * It is not possible to pass arguments to the maven command (for example, a specific Maven profile)
* **Node**:
  * Custom build steps (e.g. executing the "build" script of package.json) are not supported.
  * Existing `node_modules` directory is deleted and dependencies reinstalled using package.json and a lockfile if present.
* **Go**
  * *(generic builder only)* Applications without a go.mod cannot have sub-packages.
  * Go 1.14 triggers a kernel bug in some versions of the Linux kernel
(versions other than 5.3.15+, 5.4.2+, or 5.5+). If using an affected version,
please set the following in your /etc/docker/daemon.json:

    ```
    "default-ulimits": {
        "memlock": {
            "Name": "memlock",
            "Soft": -1,
            "Hard": -1
        }
    },
    ```

---

## Support

Please note that this project is not an officially supported Google product.
Customers of Google Cloud can use [standard support channels](https://cloud.google.com/support-hub)
for help using buildpacks with Google Cloud Products.

----

## Get involved with the community

We welcome contributions! Here's how you can contribute:

* [Browse issues](https://github.com/GoogleCloudPlatform/buildpacks/issues) or [file an issue](https://github.com/GoogleCloudPlatform/buildpacks/issues/new)
* Contribute:
  * *Read the [contributing guide](https://github.com/GoogleCloudPlatform/buildpacks/blob/master/CONTRIBUTING.md) before starting work on an issue*
  * Try to fix [good first issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/good%20first%20issue)
  * Help out on [issues that need help](https://github.com/GoogleCloudPlatform/buildpacks/labels/help%20wanted)
  * Join in on [discussion issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/discuss)
  * Join us on [Slack](https://googlecloud-community.slack.com/archives/C011ZHLLB2T)
<!--  * Read the [style guide] -->

## License

See [LICENSE](LICENSE).


