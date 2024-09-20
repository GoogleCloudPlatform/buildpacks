# The .NET Builder
This directory contains the definition of the .NET builder.

## Build the Image
To build the builder image for ubuntu-18 builder, run:

```bash
bazel build //builders/dotnet:builder.image
```

To build the builder image for ubuntu-22 builder, run:

```bash
bazel build //builders/dotnet:builder_22.image
```

## Build a Test Application
To build the sample application [simple](../testdata/dotnet/generic/simple/), run:

```bash
pack build sample-dotnet --builder gcp/dotnet --path builders/testdata/dotnet/generic/simple/ --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/dotnet/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/dotnet/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/dotnet/acceptance:gcf_test
```
