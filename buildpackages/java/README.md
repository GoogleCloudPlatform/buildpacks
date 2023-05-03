# Google Cloud's Java Buildpack
This directory contains the definition of the Java builder.

## Package the Buildpack
To package this buildpack, run:

```bash
bazel build //buildpackages/java:buildpackage
```