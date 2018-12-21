# Contribution Guidelines

As all source{d} projects, this project follows the
[source{d} Contributing Guidelines](https://github.com/src-d/guide/blob/master/engineering/documents/CONTRIBUTING.md).


# Additional Contribution Guidelines

In addition to the [source{d} Contributing Guidelines](https://github.com/src-d/guide/blob/master/engineering/documents/CONTRIBUTING.md),
this project follows the following guidelines.


## Generated Code

Before submitting a pull request make sure all the generated code changes are also committed.


### kallax

To generate go code from [kallax](https://github.com/src-d/go-kallax) models, run:

```bash
$ go generate ./...
```

To update [go-bindata](https://github.com/jteeuwen/go-bindata) with the new migration files:

```bash
$ make dependencies
$ kallax migrate --input ./store/models/ --out ./store/migrations --name <name>
$ make bindata
```

### Dependencies

Go dependencies are managed with [dep](https://golang.github.io/dep/). Use `make godep` to make sure the `vendor` directory is up to date, and commit any necessary changes.


### TOC

Please update the readme Table of Contents with:

```bash
$ make toc
```


## Build

You can separately build the binaries provided by **source{d} Lookout**; the binaries will be stored under `build/bin` directory.

### Server

```bash
$ make build
```

### lookout-sdk

```bash
$ make -f Makefile.sdk build
```


## Testing

For unit-tests run:

```bash
$ make test
```

For `lookout-sdk` integration tests (`-short` will skip tests that require bblfsh):

```bash
$ make test-sdk
$ make test-sdk-short
```

For `lookoutd serve` integration tests:

```bash
$ make test-json
```


## Dummy Analyzer

[Dummy analyzer](/cmd/dummy/main.go) is a simple analyzer implementation example.

It is part of the **Lookout** codebase but its release cycle is managed independently from the main one.

Dummy analyzer container images will be published everytime it's created a new tag with the `dummy` prefix, e.g. `dummy-v0.0.1`

It can be built locally running:

```bash
$ make -f Makefile.dummy build
```
