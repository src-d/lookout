# Contribution Guidelines

As all source{d} projects, this project follows the
[source{d} Contributing Guidelines](https://github.com/src-d/guide/blob/master/engineering/documents/CONTRIBUTING.md).


## Additional Contribution Guidelines

In addition to the [source{d} Contributing Guidelines](https://github.com/src-d/guide/blob/master/engineering/documents/CONTRIBUTING.md),
this project follows the guidelines described below.


# Development 

## Build

You can separatelly build the binaries provided by **lookout**; the binaries will be stored under `build/bin` directory.

**server**:
```bash
$ make build
```

**lookout-sdk**:
```bash
$ make -f Makefile.sdk build
```

**dummy** analyzer:
```bash
$ make -f Makefile.dummy build
```

## Code generation

To generate go code from models, run:

```bash
$ go generate ./...
```

To update [go-bindata](https://github.com/jteeuwen/go-bindata) with the new migration files:

```bash
$ kallax migrate --input ./store/models/ --out ./store/migrations --name <name>
$ make dependencies
$ make bindata
```

## Testing

For unit-tests run:
```bash
$ make test
```

For SDK integration tests:
```bash
$ make test-sdk
```

For lookout serve integration tests:
```bash
$ make test-json
```

## Dummy Analyzer Release

To publish the dummy analyzer container you need to create a tag with the `dummy` prefix, e.g. `dummy-v0.0.1`. Please note this this doesn't require to do a GitHub release, we just need the Git tag.

A normal release tag will not publish this container.
