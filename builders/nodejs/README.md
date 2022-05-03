# The Node.js Builder
This directory contains the definition of the Node.js builder.

## Build the Image
To build the builder image, run:

```bash
bazel build //builders/nodejs:builder.image
```

## Build a Test Application
To build the sample application [gomod_go_sum](../testdata/nodejs/package_json/), run:

```bash
pack build sample-nodejs --builder gcp/nodejs --path builders/testdata/nodejs/package_json/ --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
bazel test //builders/nodejs/acceptance:acceptance_test
```

### Test a Single Product
For each product, there exists a suite of tests. The build target is
`builders/go/acceptance:<product>_test`. Where `<product>` is replaced with the
acronym of the product. For example, to run the tests for Google Cloud
Functions, run:

```bash
bazel test //builders/nodejs/acceptance:gcf_test
```
