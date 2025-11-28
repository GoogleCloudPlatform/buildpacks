# Google Cloud's buildpacks

This repository contains a set of builders and buildpacks designed to run on
Google Cloud's container platforms:
 [Cloud Run](https://cloud.google.com/run),
 [GKE](https://cloud.google.com/kubernetes-engine),
 [Anthos](https://cloud.google.com/anthos),
 and [Compute Engine running Container-Optimized OS](https://cloud.google.com/container-optimized-os/docs).
 They are also used as the build system for [App Engine](https://cloud.google.com/appengine) and [Cloud Functions](https://cloud.google.com/functions).
 They are 100% compatible with [Cloud Native Buildpacks](https://buildpacks.io/).

## To start using Google Cloud's buildpacks

* Read our documentation on [Google Cloud's buildpacks](https://cloud.google.com/docs/buildpacks/overview)
* Try [building an application](https://cloud.google.com/docs/buildpacks/build-application) or [building a function](https://cloud.google.com/docs/buildpacks/build-function) with Google Cloud's buildpacks

## Additional tooling

The Google Cloud's buildpacks project provides builder images suitable for use
with
[pack](https://github.com/buildpacks/pack),
[kpack](https://github.com/pivotal/kpack),
[tekton](https://github.com/tektoncd/catalog/tree/HEAD/task/buildpacks/0.1),
[skaffold](https://github.com/GoogleContainerTools/skaffold/tree/HEAD/examples/buildpacks),
and other tools that support the Buildpacks v3 specification.

## Additional Configurations
Google Cloud's buildpacks can be configured in a few ways:

* [Setting Environment Variables](https://cloud.google.com/docs/buildpacks/set-environment-variables)
* [Google Cloud-specific configurations](https://cloud.google.com/docs/buildpacks/service-specific-configs)
* [Custom Build and Run images](https://cloud.google.com/docs/buildpacks/build-run-image)
* Language specific configurations for:
    * [Node.js](https://cloud.google.com/docs/buildpacks/nodejs)
    * [Python](https://cloud.google.com/docs/buildpacks/python)
    * [Go](https://cloud.google.com/docs/buildpacks/go)
    * [Java](https://cloud.google.com/docs/buildpacks/java)
    * [Ruby](https://cloud.google.com/docs/buildpacks/ruby)

## App Engine and Cloud Function Builders and Buildpacks

These builders create container images designed to run on Google Cloud's App
Engine and Functions services. Most of the buildpacks are
identical to those in the general builder.

Compared to the general builder, there are two primary differences. First,
there are additional buildpacks which add transformations specific to each
service. Second, in order to optimize execution speed, each
language has a separate builder.

As an example, in order to build a Docker container image  for Google App Engine
Java17 runtime you can use:

```bash
pack build <app-name>  --builder gcr.io/serverless-runtimes/google-22-full/builder/java
```

If you rely on a custom App Engine entrypoint in your app.yaml, you can use:

```bash
pack build <app-name>  --builder gcr.io/serverless-runtimes/google-22-full/builder/java  --env GOOGLE_ENTRYPOINT="your entry point command"
```

The application container image can then be executed locally:

```bash
docker run --rm -p 8080:8080 <app-name>
```
Locally, your application might depend on App Engine [environment variables](https://cloud.google.com/appengine/docs/standard/java-gen2/runtime#environment_variables) that would need to be set in the local environment.

## Learn more about Cloud Native Buildpacks

This project implements the Cloud Native Buildpacks specification. 
To read more, see Cloud Native Buildpacks project
[documentation](https://buildpacks.io/docs/for-app-developers/concepts/).

For those new to buildpacks, these concepts are good starting points:

* **[Builder](https://buildpacks.io/docs/concepts/components/for-app-developers/builder/)** A container image that contains buildpacks and detection order in which builds are executed.
* **[Buildpack](https://buildpacks.io/docs/concepts/components/for-app-developers/buildpack/)** An executable that "inspects your app source code and formulates a plan to build and run your application".
* **Buildpack Group** Several buildpacks which together provide support for a
specific language or framework.
* **[Run Image](https://buildpacks.io/docs/for-app-developers/concepts/base-images/stack/)** The container image that serves as the base for the built application.

## Support

Google Cloud's buildpacks are only officially supported when used with Google Cloud products.
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


