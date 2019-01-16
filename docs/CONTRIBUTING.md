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

### lookout-sdk

```shell
$ make -f Makefile.sdk build
```


## Testing

For unit-tests run:

```shell
$ make test
```

For `lookout-sdk` integration tests (`-short` will skip tests that require bblfsh):

```shell
$ make test-sdk
$ make test-sdk-short
```

For `lookoutd serve` integration tests:

```shell
$ make test-json
```


## Web Interface

If you want to contribute in **source{d} Lookout** [Web Interface](web.md), you should consider the following:

### Dependencies

[Node.js](https://nodejs.org) `>=8` is required; you can check which version of `node` do you have, running:

```shell
$ node -v
v10.15.0
```

### Running

In case you want to locally run the web interface from sources, you can choose one of the following:
- using web assets from binaries (embeded by `esc`), that will require you to restart the server everytime you change any web asset.
    ```shell
    $ make -f Makefile.web web-serve
    ```
    And navigate to [http://127.0.0.1:8080](http://127.0.0.1:8080)
- using `create-react-app` dev server, with live reload for web assets changes, running in separated terminals the backend: 
    ```shell
    $ make -f Makefile.web web-start
    ```
    and the frontend:
    ```shell
    $ make -f Makefile.web web-dependencies # if you didn't do it yet
    $ yarn --cwd frontend start
    ```
    Configure the GitHub App authorization callback URL to `http://127.0.0.1:3000/callback`, and navigate to [http://127.0.0.1:3000](http://127.0.0.1:3000)

### Testing

For unit-tests over the Web Interface frontend:

```shell
$ make -f Makefile.web web-dependencies # if you didn't do it yet
$ make -f Makefile.web web-test
```


## dummy Analyzer

[`dummy` analyzer](analyzers-examples.md#dummy-analyzer) is part of the **source{d} Lookout** codebase but its release cycle is managed independently from the main one.

`dummy` analyzer container images will be published everytime it's created a new tag with the `dummy` prefix, e.g. `dummy-v0.0.1`

It can be built locally running:

```shell
$ make -f Makefile.dummy build
```
