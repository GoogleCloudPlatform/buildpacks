# Google Cloud Platform Buildpacks

Google Cloud Platform Buildpacks is a set of buildpacks and builder definitions
based on the Buildpack v3 [specification](https://github.com/buildpacks/spec).

Supported languages:

* Go (1.12+)
* Java (11)
* Node.js (10+)
* PHP (7.3+)
* Python (3.7+)
* Ruby (2.5+)
* .NET (3+)

TODO: Add a grid of buildpacks per language and product.

Features:

* Builders are compatible with [pack](https://github.com/buildpacks/pack).
* Builders are used in App Engine and Cloud Functions.

For more details on Cloud Native Buildpacks, please visit https://buildpacks.io

----

Note: Go 1.14 triggers a kernel bug in some versions of the Linux kernel
(version other than 5.3.15+, 5.4.2+, or 5.5+). If using an affected version,
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

----

## Usage

The GCP Buildpacks project provides builder images suitable for use
with [`pack`](https://github.com/buildpacks/pack). Most users will be
interested in the `gcr.io/buildpacks/builder` builder.

### Building an application

The following command invokes `pack` to apply the GCP builder to build
the application in the current directory, and then containerize the result
into a container image named `<app>`.

```bash
pack build <app> --builder gcr.io/buildpacks/builder
```

### Runtime configuration

GCP Buildpacks support configuration using a set of environment
variables that are supported accross runtimes.

* `GOOGLE_DEVMODE`
  * Enables the development mode buildpack.
  * **Example**: `true`, `True`, `1` will enable development mode.
* `GOOGLE_ENTRYPOINT`:
  * Specifies entrypoint to set on the final image.
  * **Example**: `gunicorn -p :8080 main:app` for Python.

Only applicable to compiled languages:

* `GOOGLE_BUILDABLE`:
  * Specifies path to a buildable unit.
  * **Example**: `./maindir` for Go will build the package rooted at maindir.

Only applicable to the `runtime` buildpacks:

* `GOOGLE_RUNTIME`:
  * If specified, forces the runtime to opt-in.
  * **Note**: If the runtime buildpack appears in multiple groups, the first group
    will be chosen consistently with the specification.
  * **Example**: `nodejs` will cause the nodejs/runtime buildpack to opt-in.
* `GOOGLE_RUNTIME_VERSION`:
  * If specified, overrides the runtime version to install.
  * **Example**: `13.7.0` for Node.js, `1.14.1` for Go.

## Get involved with the community

We welcome contributions! Here's how you can contribute:

* [Browse issues](https://github.com/GoogleCloudPlatform/buildpacks/issues) or [file an issue](https://github.com/GoogleCloudPlatform/buildpacks/issues/new)
* Contribute:
  * *Read the [contributing guide](https://github.com/GoogleCloudPlatform/buildpacks/blob/master/CONTRIBUTING.md) before starting work on an issue*
  * Try to fix [good first issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/good%20first%20issue)
  * Help out on [issues that need help](https://github.com/GoogleCloudPlatform/buildpacks/labels/help%20wanted)
  * Join in on [discussion issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/discuss)
<!--  * Read the [style guide] -->

Please note that this project is not an officially supported Google product.

## License

See [LICENSE](LICENSE).


