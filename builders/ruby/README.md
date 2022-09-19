
# The Ruby Builder
This directory contains the definition of the Ruby builder.

## Build the Image
To build the builder image, run:

```bash
bazel build //builders/ruby:builder.image
```

## Build a Test Application
To build the sample application [simple](../testdata/ruby/generic/simple/), run:

```bash
pack build sample-ruby --builder gcp/ruby --path builders/testdata/ruby/generic/simple/ --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/ruby/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/ruby/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/ruby/acceptance:gcf_test
```