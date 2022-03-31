# The Go Builder
This directory contains the definition of the Go builder.

## Build the Image
To build the builder image, run:

```bash
bazel build builders/go:builder.image
```

## Build a Test Application
To build the sample application [gomod_go_sum](../testdata/go/gomod_go_sum/), run:

```bash
pack build sample-go --builder gcp/go --path builders/testdata/go/gomod_go_sum/ --trust-builder -v
```
