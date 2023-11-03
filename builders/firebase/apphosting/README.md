# The Firebase App Hosting Builder
This directory contains the definition of the Firebase App Hosting builder.

IMPORTANT: Firebase "App Hosting" is a new product name from Firebase and unrelated to "apphosting" a.k.a. App Engine.

## Build the Image
To build the builder image, run:

```bash
$ bazel build //builders/firebase/apphosting:firebase_app_hosting_22_builder.image
```

## Build a Test Application
To build the sample application [generic/simple](../../testdata/nodejs/generic/simple/), run:

```bash
$ pack build sample-nodejs --builder firebase/apphosting --path builders/testdata/nodejs/generic/simple/ --trust-builder -v
```

## Acceptance Tests
To run the acceptance tests across all the products, run:

```bash
$ bazel test //builders/firebase/apphosting/acceptance:nodejs_test
```