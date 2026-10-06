# Google Cloud Label Image Buildpack

The label-image buildpack adds any environment variables with the
`GOOGLE_LABEL_` prefix as labels in the final application image.

## Usage

Compile and package the buildpack using [Bazel](https://bazel.build/):

```bash
bazel build cmd/utils/label:label.tgz
```

This will create a tgz archive in the `/bazel-bin` directory that you
can use to build an application with the
[pack cli](https://buildpacks.io/docs/tools/pack/).

```bash
pack build label-test \
  --path builders/testdata/nodejs/package_json \
  --buildpack bazel-bin/cmd/utils/label/label.tgz \
  --env="GOOGLE_LABEL_FOO=bar"
```

Any build-time environment variables with the prefix `GOOGLE_LABEL` will be 
added to the run image as labels with the `google.` prefix:

```bash
docker inspect --format='{{index .Config.Labels "google.foo"}}' label-test
> bar
```

## Testing

You can run all unit tests with:

```
bazel test cmd/utils/label/...
```

## Contributing

Please see our [contributing guide](../../../CONTRIBUTING.md).