# The Java Builder
This directory contains the definition of the Java builder.

## Build the Image
To build the builder image for ubuntu-18 builder, run:

```bash
bazel build //builders/java:builder.image
```

To build the builder image for ubuntu-22 builder, run:

```bash
bazel build //builders/java:builder_22.image
```

To build the builder image for ubuntu-24 builder, run:

```bash
bazel build //builders/java:builder_24.image
```

## Build a Test Application
To build the sample application [http-server](../testdata/java/appengine/http-server), run:

```bash
pack build sample-java --builder gcp/java --path <path to java app> --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/java/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/go/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/java/acceptance:gcf_test
```