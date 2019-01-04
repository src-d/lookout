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

```shell
$ go generate ./...
```

To update embedded migrations with the new files:

```shell
$ make dependencies
$ kallax migrate --input ./store/models/ --out ./store/migrations --name <name>
$ make pack-migrations
```

### Dependencies

Go dependencies are managed with [dep](https://golang.github.io/dep/). Use `make godep` to make sure the `vendor` directory is up to date, and commit any necessary changes.


### TOC

Please update the readme Table of Contents with:

```shell
$ make toc
```


## Build

You can separately build the binaries provided by **source{d} Lookout**; the binaries will be stored under `build/bin` directory.

### Server

```shell
$ make build
```

### lookout-tool

```shell
$ make -f Makefile.tool build
```


## Testing

For unit-tests run:

```shell
$ make test
```

For `lookout-tool` integration tests (`-short` will skip tests that require bblfsh):

```shell
$ make test-tool
$ make test-tool-short
```

For `lookoutd serve` integration tests:

```shell
$ make test-json
```


## dummy Analyzer

[`dummy` analyzer](analyzers-examples.md#dummy-analyzer) is part of the **source{d} Lookout** codebase but its release cycle is managed independently from the main one.

`dummy` analyzer container images will be published everytime it's created a new tag with the `dummy` prefix, e.g. `dummy-v0.0.1`

It can be built locally running:

```shell
$ make -f Makefile.dummy build
```
