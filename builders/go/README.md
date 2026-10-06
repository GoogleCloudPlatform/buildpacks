# The Go Builder
This directory contains the definition of the Go builder.

## Build the Image
To build the builder image for ubuntu-18 builder, run:

```bash
bazel build //builders/go:builder.image
```

To build the builder image for ubuntu-22 builder, run:

```bash
bazel build //builders/go:builder_22.image
```

To build the builder image for ubuntu-24 builder, run:

```bash
bazel build //builders/go:builder_24.image
```

## Build a Test Application
To build the sample application [gomod_go_sum](../testdata/go/gomod_go_sum/), run:

```bash
pack build sample-go --builder gcp/go --path builders/testdata/go/gomod_go_sum/ --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/go/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/go/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/go/acceptance:gcf_test
```
