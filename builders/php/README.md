# The PHP Builder
This directory contains the definition of the PHP builder.

## Build the Image
To build the builder image, run:

```bash
bazel build //builders/php:builder.image
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/php/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/php/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/php/acceptance:gcf_test
```