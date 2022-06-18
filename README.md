# Google Cloud Buildpacks

This repository contains a set of builders and buildpacks designed to run on
Google Cloud's container platforms:
 [Cloud Run](https://cloud.google.com/run),
 [GKE](https://cloud.google.com/kubernetes-engine),
 [Anthos](https://cloud.google.com/anthos),
 and [Compute Engine running Container-Optimized OS](https://cloud.google.com/container-optimized-os/docs).
 They are also used as the build system for [App Engine](https://cloud.google.com/appengine) and [Cloud Functions](https://cloud.google.com/functions).
 They are 100% compatible with [Cloud Native Buildpacks](https://buildpacks.io/).

## Quickstart

1. [Install Docker](https://store.docker.com/search?type=edition&offering=community)
2. [Install the pack tool (a CLI for running Buildpacks)](https://buildpacks.io/docs/install-pack/)
3. Clone the [sample apps](https://github.com/GoogleCloudPlatform/buildpack-samples):

    ```
    git clone https://github.com/GoogleCloudPlatform/buildpack-samples.git
    cd buildpack-samples
    ```

4. Pick a sample and build it, for instance with `sample-go`:

    ```
    cd sample-go
    pack build --builder gcr.io/buildpacks/builder:v1 sample-go
    ```

5. Run it with docker, like:

    ```
    docker run --rm -p 8080:8080 sample-go
    ```
   
6. See it in action

    ```
    curl localhost:8080
    # Output: hello, world
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


## General Builder and Buildpacks

This is a general purpose builder that creates container images designed to run on most
platforms (e.g. Kubernetes / Anthos, Knative / Cloud Run, Container OS, etc),
and should be used by the majority of users. The builder attempts to autodetect
the language of your source code, and can also build functions compatible with
the [Google Cloud Function Framework](https://cloud.google.com/functions/docs/functions-framework) by [setting the GOOGLE_FUNCTION_TARGET env var](#configuration).

The general builder is hosted at `gcr.io/buildpacks/builder:v1`.

Supported languages include:


|Runtime            |App Support | Function Support  |
|-------------------|:----------:|:-----------------:|
| Go 1.10 +         | ✓          | ✓                 |
| Node.js 10 +      | ✓          | ✓                 |
| Python 3.7 +      | ✓          | ✓                 |
| Java 8 +          | ✓          | ✓ (11 + only)     |
| .NET Core 3.1 +   | ✓          | ✓                 |
| Ruby 2.6 +        | ✓          | ✓                 |

For Ruby functions, the entrypoint has to be set manually (as seen in the [sample apps](https://github.com/GoogleCloudPlatform/buildpack-samples)).

## App Engine and Cloud Function Builders and Buildpacks

These builders create container images designed to run on Google Cloud's App
Engine and Functions services. Most of the buildpacks are
identical to those in the general builder.

Compared to the general builder, there are two primary differences. First,
there are additional buildpacks which add transformations specific to each
service. Second, in order to optimize execution speed, each
language has a separate builder.

## Usage

The Google Cloud Buildpacks project provides builder images suitable for use
with
[pack](https://github.com/buildpacks/pack),
[kpack](https://github.com/pivotal/kpack),
[tekton](https://github.com/tektoncd/catalog/tree/HEAD/task/buildpacks/0.1),
[skaffold](https://github.com/GoogleContainerTools/skaffold/tree/HEAD/examples/buildpacks),
and other tools that support the Buildpacks v3 specification.


### Building an application

The following command invokes [pack](https://github.com/buildpacks/pack) to
apply the general builder to build the application in the current directory, and then
containerizes the result into a local container image named `<app-name>`.

```bash
pack build <app-name> --builder gcr.io/buildpacks/builder:v1
```

The application you built can then be executed locally:

```bash
docker run --rm -p 8080:8080 <app-name>
```

You can set Cloud Buildpacks as your default:

```bash
pack set-default-builder gcr.io/buildpacks/builder:v1
```

And you can publish the built image to the cloud directly with [pack](https://github.com/buildpacks/pack):

```bash
pack build --publish gcr.io/YOUR_PROJECT_ID/APP_NAME
```

### Building a function

The same commands as above can be used to build a function image. The following command builds
a function called `myFunction` and produces a local image named `<fn-name>`.

```bash
pack build <fn-name> --builder gcr.io/buildpacks/builder:v1 --env GOOGLE_FUNCTION_TARGET=myFunction
```

### Extending the run image

If your application requires additional system packages to be installed and
available when it runs, you can accomplish this by customizing the **run**
container image.

```bash
cat > run.Dockerfile << EOF
FROM gcr.io/buildpacks/gcp/run:v1
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
pack build my-app --builder gcr.io/buildpacks/builder:v1 --run-image my-run-image
```
### Extending the builder image

If you require certain packages for **building** your application, create a custom
builder image based on the base builder:

```bash
cat > builder.Dockerfile << EOF
FROM gcr.io/buildpacks/builder:v1
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
variables** that are set at build time. For example, when using `pack`:

```bash
pack build my-app --builder gcr.io/buildpacks/builder:v1 --env GOOGLE_ENTRYPOINT="gunicorn -p :8080 main:app"
```

#### Common options

The following confguration options are supported across runtimes:

* `GOOGLE_ENTRYPOINT`
  * Specifies the command which is run when the container is executed; equivalent to [entrypoint](https://docs.docker.com/engine/reference/builder/#entrypoint) in a Dockerfile.
  * See the [default entrypoint behavior](#default-entrypoint-behavior) section for default behavior.
  * **Example:** `gunicorn -p :8080 main:app` for Python. `java -jar target/myjar.jar` for Java.
* `GOOGLE_RUNTIME`
  * If specified, forces the runtime to opt-in. If the runtime buildpack appears in multiple groups, the first group will be chosen, consistent with the buildpack specification.
  * *(Only applicable to buildpacks install language runtime or toolchain.)*
  * **Example:** `nodejs` will cause the nodejs/runtime buildpack to opt-in.
* `GOOGLE_RUNTIME_VERSION`
  * If specified, overrides the runtime version to install. In .NET, overrides the .NET SDK version to install.
  * *(Only applicable to buildpacks install language runtime or toolchain.)*
  * **Example:** `13.7.0` for Node.js, `1.14.1` for Go, `8` for Java, `3.1.301` for .NET.
* `GOOGLE_BUILDABLE`
  * Specifies path to a buildable unit.
  * *(Only applicable to .NET, Dart and Go languages.)*
  * **Example:** `./maindir` for Go will build the package rooted at maindir.
* `GOOGLE_BUILD_ARGS`
  * Appends arguments to build command.
  * *(Currently only applicable to Java Maven and Gradle and .NET)*
  * **Example:** `-Pprod` for a Java will run `mvn clean package ... -Pprod`.
* `GOOGLE_DEVMODE`
  * Enables the development mode buildpacks. This is used by [Skaffold](https://skaffold.dev) to enable live local development where changes to your source code trigger automatic container rebuilds. To use, install Skaffold and run `skaffold dev`.
  * **Example:** `true`, `True`, `1` will enable development mode.
* `GOOGLE_CLEAR_SOURCE`
  * Clears source after the application is built. If the application depends on static files, such as Go templates, setting this variable may cause the application to misbehave.
  * *(Only applicable to Go apps and Java apps & functions.)*
  * **Example:** `true`, `True`, `1` will clear the source.

Certain buildpacks support other environment variables:

#### Functions Framework buildpacks

For use with source code built around the [Google Cloud Functions Framework](https://cloud.google.com/functions/docs/functions-framework).
See the [contract](https://github.com/GoogleCloudPlatform/functions-framework) for more information about the configuration options.

* `GOOGLE_FUNCTION_TARGET`
  * Specifies the name of the exported function to be invoked in response to requests.
  * **Example:** `myFunction` will cause the Functions Framework to invoke the function of the same name.
* `GOOGLE_FUNCTION_SIGNATURE_TYPE`
  * Specifies the signature used by the function.
  * **Example:** `http`, `event`, or `cloudevent`.
* `GOOGLE_FUNCTION_SOURCE`
  * Specifies the name of the directory or file containing the function source, depending on the language.
  * *(Only applicable to some languages, please see the language-specific [documentation](https://github.com/GoogleCloudPlatform/functions-framework#languages).)*
  * **Example:** `function.py` for Python.

#### Go Buildpacks

* `GOOGLE_GOGCFLAGS`
  * Passed to `go build` and `go run` as `-gcflags value` with no interpretation.
  * **Example:** `all=-N -l` enables race condition analysis and changes how source filepaths are recorded in the binary.
* `GOOGLE_GOLDFLAGS`
  * Passed to `go build` and `go run` as `-ldflags value` with no interpretation.
  * **Example:** `-s -w` is used to strip and reduce binary size.

#### Language-idiomatic configuration options

Buildpacks support language-idiomatic configuration through environment
variables. These environment variables should be specified without a
`GOOGLE_` prefix.

* **Go**
  * `GO<key>`, see [documentation](https://golang.org/cmd/go/#hdr-Environment_variables).
      * **Example:** `GOFLAGS=-flag=value` passes `-flag=value` to `go` commands.
* **Java**
  * `MAVEN_OPTS`, see [documentation](https://maven.apache.org/configure.html).
      * **Example:** `MAVEN_OPTS=-Xms256m -Xmx512m` passes these flags to the JVM running Maven.
      * **Example:** `MAVEN_OPTS=--add-opens java.base/java.lang=ALL-UNNAMED` to suppress "illegal reflective access" warnings from Maven.
  * `GRADLE_OPTS`, see [documentation](https://docs.gradle.org/current/userguide/build_environment.html#sec:gradle_configuration_properties).
      * **Example:** `GRADLE_OPTS=-Xms256m -Xmx512m` passes these flags to the JVM running Gradle.
  * Using the Google Maven Central mirror
      * Mirror URLs can be found [here](https://storage-download.googleapis.com/maven-central/index.html).
      * For Maven: [Using Mirrors for Repositories](https://maven.apache.org/guides/mini/guide-mirror-settings.html)
         * Create a copy of the settings.xml from the default location of ~/.m2/settings.xml to inside your application source directory and specify `GOOGLE_BUILD_ARGS=--settings <path/to/settings>`.
         * Note: <path/to/settings> is relative to the source directory.
         * Example `settings.xml`:

              ```
              <settings>
                <mirrors>
                  <mirror>
                    <id>google-maven-central</id>
                    <name>GCS Maven Central mirror</name>
                    <url>https://maven-central.storage-download.googleapis.com/maven2/</url>
                    <mirrorOf>central</mirrorOf>
                  </mirror>
                </mirrors>
              </settings>
              ```
      * For Gradle: [Declaring Repositories](https://docs.gradle.org/current/userguide/declaring_repositories.html)
          * Example `build.gradle` entry:

              ```
              repositories {
                  maven {
                      url "https://maven-central.storage-download.googleapis.com/maven2/"
                  }
              }
              ```
  * The Maven buildpack supports the [Polyglot extensions for Maven](https://github.com/takari/polyglot-maven#overview).
    The Polyglot extensions are enabled by placing a suitably-configured
    `.mvn/extensions.xml` in your application source directory.
* **Node.js**
  * `NPM_CONFIG_<key>`, see [documentation](https://docs.npmjs.com/misc/config#environment-variables).
      * **Example:** `NPM_CONFIG_FLAG=value` passes `-flag=value` to `npm` commands.
* **PHP**
  * `COMPOSER_<key>`, see [documentation](https://getcomposer.org/doc/03-cli.md#environment-variables).
      * **Example:** `COMPOSER_PROCESS_TIMEOUT=60` sets the timeout for `composer` commands.
* **Python**
  * `PIP_<key>`, see [documentation](https://pip.pypa.io/en/stable/user_guide/#environment-variables).
      * **Example:** `PIP_DEFAULT_TIMEOUT=60` sets `--default-timeout=60` for `pip` commands.
* **Ruby**
  * `BUNDLE_<key>`, see [documentation](https://bundler.io/v2.0/bundle_config.html#LIST-OF-AVAILABLE-KEYS).
      * **Example:** `BUNDLE_TIMEOUT=60` sets `--timeout=60` for `bundle` commands.


## Default entrypoint behavior

* If `GOOGLE_ENTRYPOINT` is specified, use:
  * `/bin/bash -c <entrypoint>`
* If `Procfile` exists at the application root and contains a `web` process, use:
  * `/bin/bash -c <web process>`
* Otherwise, use language-specific behavior below.

### Language-specific behavior

* **.NET**
  * Search the output directory for a binary or a library with the same name as the project file (e.g. app.csproj --> app or app.dll).
      * `/bin/bash -c exec <project name>`
      * `/bin/bash -c dotnet <project name>.dll`
  * Parse the project file for an `AssemblyName` field and search for the associated binary or library file in the output directory.
      * `/bin/bash -c exec <assembly name>`
      * `/bin/bash -c dotnet <assembly name>.dll`
* **Go**
  * Use `<layer/path>/main`, where `main` is the compiled binary.
  * The `main` binary is also available on `$PATH`.
* **Java**
  * Use the first executable .jar file found in the following directories, in order:
      * `<workspace>/target`
      * `<workspace>/build`
      * `<workspace>/build/libs`
      * `<workspace>`
  * If exactly one executable .jar file is found, use:
      * `java -jar <executable.jar>`
  * For "exploded jars", e.g. Spring Boot, find the `Main-Class` entry from the manifest and use:
      * `java -classpath . <class>`
* **Node.js**
  * Use `npm start`; see the [npm documentation](https://docs.npmjs.com/cli/start.html).
* **PHP**
  * Not available in the general builder.
* **Python**
  * No default entrypoint logic.
* **Ruby**
  * No default entrypoint logic.

## Known Limitations

* **General**:
  * Caching is project-specific, not cross-project. Dependencies, such as the JDK, cannot be shared across projects and need to be redownloaded on first build.
  * Built images have their creation time set to 40 years in the past. See [reproducible builds](https://buildpacks.io/docs/reference/reproducibility/).
* **Node**:
  * Custom build steps (e.g. executing the "build" script of package.json) are not supported.
  * Existing `node_modules` directory is deleted and dependencies reinstalled using package.json and a lockfile if present.
* **Python**
  * Private dependencies must be vendored. The build does not have access to private repository credentials and cannot pull dependencies at build time.
    Please see the App Engine [instructions](https://cloud.google.com/appengine/docs/standard/python3/specifying-dependencies#private_dependencies).
* **Go**
  * Private dependencies must be vendored. The build does not have access to private repository credentials and cannot pull dependencies at build time.
    Please see the App Engine [instructions](https://cloud.google.com/appengine/docs/standard/go/specifying-dependencies#using_private_dependencies)
  * *(general builder only)* Applications without a go.mod cannot have sub-packages.
  * Go 1.14 triggers a kernel bug in some versions of the Linux kernel
(versions other than 5.3.15+, 5.4.2+, or 5.5+). If using an affected version,
please set the following in your `/etc/docker/daemon.json`:

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

## Using with Google Cloud Build

The buildpack builder can be invoked as a step of a [Google Cloud Build](https://cloud.google.com/cloud-build) process, for instance by using the pack builder image provided by the Skaffold project:

```
steps:
- name: 'gcr.io/k8s-skaffold/pack'
  entrypoint: 'pack'
  args: ['build', '--builder=gcr.io/buildpacks/builder', '--publish', 'gcr.io/$PROJECT_ID/sample-go:$COMMIT_SHA']
```

There is also support for invoking this builder directly [using `gcloud`](https://cloud.google.com/sdk/gcloud/reference/builds/submit):

```
gcloud builds submit --pack image=gcr.io/my-project/imageg
```

This command will send the local source directory to Cloud Build, invoke this buildpack builder on it, and publish the resulting image to Google Container Registry.

## Support

Google Cloud Buildpacks are only officially supported when used with Google Cloud products.
Customers of Google Cloud can use [standard support channels](https://cloud.google.com/support-hub)
for help using buildpacks with Google Cloud Products.

## Security

For information on reporting security vulnerabilities, see [SECURITY.md](./SECURITY.md).

## Get involved with the community

We welcome contributions! Here's how you can contribute:

* [Browse issues](https://github.com/GoogleCloudPlatform/buildpacks/issues) or [file an issue](https://github.com/GoogleCloudPlatform/buildpacks/issues/new)
* Contribute:
  * *Read the [contributing guide](https://github.com/GoogleCloudPlatform/buildpacks/blob/main/CONTRIBUTING.md) before starting work on an issue*
  * Try to fix [good first issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/good%20first%20issue)
  * Help out on [issues that need help](https://github.com/GoogleCloudPlatform/buildpacks/labels/help%20wanted)
  * Join in on [discussion issues](https://github.com/GoogleCloudPlatform/buildpacks/labels/discuss)
<!--  * Read the [style guide]  -->

## License

See [LICENSE](LICENSE).


