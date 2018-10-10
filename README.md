lookout
[![Build Status](https://travis-ci.org/src-d/lookout.svg)](https://travis-ci.org/src-d/lookout)
[![GoDoc](https://godoc.org/gopkg.in/src-d/lookout?status.svg)](https://godoc.org/github.com/src-d/lookout)
[![Code Coverage](https://codecov.io/github/src-d/lookout/coverage.svg)](https://codecov.io/github/src-d/lookout)
[![Go Report Card](https://goreportcard.com/badge/github.com/src-d/lookout)](https://goreportcard.com/report/github.com/src-d/lookout)
![development](https://svg-badge.appspot.com/badge/stability/development?color=D6604A)
=======

A service for assisted code review, that allows running custom code Analyzers on pull requests.

# Table of Contents

<!--ts-->
   * [Configuring lookout](#configuring-lookout)
   * [Usage](#usage)
      * [Running lookout with docker-compose](#running-lookout-with-docker-compose)
      * [Running lookout from Binaries](#running-lookout-from-binaries)
         * [Installation](#installation)
         * [Dependencies](#dependencies)
         * [Quickstart](#quickstart)
      * [Running in Distributed Mode](#running-in-distributed-mode)
   * [Available Analyzers](#available-analyzers)
   * [SDK for Analyzer Developers](#sdk-for-analyzer-developers)
   * [Development](#development)
      * [Build](#build)
      * [Code generation](#code-generation)
      * [Testing](#testing)
      * [Dummy Analyzer Release](#dummy-analyzer-release)
   * [Contribute](#contribute)
   * [Code of Conduct](#code-of-conduct)
   * [License](#license)
<!--te-->

# Configuring lookout

Please refer to the [**Configuring lookout guide**](docs/configuration.md) for documentation for the `config.yml` file.

# Usage

## Running lookout with docker-compose

Using [Docker Compose](https://docs.docker.com/compose) you can use the provided [`docker-compose.yml`](docker-compose.yml) config file to start **lookout**, its dependencies (**bblfsh** and **PostgreSQL**) and the `dummy` analyzer which will add some stats to the watched pull requests.

To do so, clone this repository or download [`docker-compose.yml`](docker-compose.yml).

Create the `config.yml` file in the same directory where `docker-compose.yml` is, and run:

```bash
$ docker-compose pull
$ GITHUB_USER=<user> GITHUB_TOKEN=<token> docker-compose up --force-recreate
```

If you need to restart the database to a clean state, you can do so with:
```bash
$ docker rm lookout_postgres_1
```

## Running lookout from Binaries

### Installation

Go to the [lookout releases page](https://github.com/src-d/lookout/releases) and download the latest `lookoutd` and `dummy` binaries from there.

### Dependencies

**lookout** needs a running instance of:

* [bblfshd](https://github.com/bblfsh/bblfshd) to parse files into [UAST](https://doc.bblf.sh/uast/uast-specification.html).
* [PostgreSQL](https://www.postgresql.org).

You will also need to configure external Analyzers, that will perform the actual processing of the pull requests. You may use the included `dummy` Analyzer for testing purposes.

### Quickstart

Following these steps you will be able to run separately the **lookout** dependencies, analyzers and the server itself.

1. Run the [dependencies](#dependencies) manually or using docker-compose, executing:

    ```bash
    $ docker-compose up bblfsh postgres
    ```

1. Initialize the database.<br />
    This command will work for the PostgreSQL created by docker-compose, use `-h` to see other options.

    ```bash
    $ lookoutd migrate
    ```

1. Start an analyzer before running **lookout**.<br />
    You can use the *dummy* one as it is provided by this repository; to do so just run:

    ```bash
    $ dummy serve
    ```

1. Copy the [`config.yml.tpl`](config.yml.tpl) into `config.yml` and add the URLs of the repositories to be watched. Take a look at [configuration and GitHub authentication](docs/configuration.md) for more details about **lookout** configuration.

1. Start **lookout** server<br />
    If you want to post the analysis results on GitHub, run:

    ```bash
    $ lookoutd serve --github-token <token> --github-user <user>
    ```

    If you want to avoid posting the analysis results on GitHub, and only print them, run:

    ```bash
    $ lookoutd serve --dry-run
    ```

## Running in Distributed Mode

_Note_: This is a work in progress feature.

Please refer to the [**Running in Distributed Mode guide**](docs/distributed.md) for documentation on the advanced distributed deployment.

# Available Analyzers

This is a list of the available analyzers for lookout:

| Name | Description | Maturity level |
| -- | -- | -- |
| [style-analyzer](https://github.com/src-d/style-analyzer) | Code style analyzer | development |
| [gometalint](https://github.com/src-d/lookout-gometalint-analyzer) | Reports [gometalinter](https://github.com/alecthomas/gometalinter) results on pull requests | testing and demo |
| [sonarcheck](https://github.com/src-d/lookout-sonarcheck-analyzer) | An analyzer that uses [bblfsh UAST](https://doc.bblf.sh/uast/uast-specification.html) and [sonar-checks](https://github.com/bblfsh/sonar-checks) to process pull requests | testing and demo |
| [terraform](https://github.com/meyskens/lookout-terraform-analyzer) | An analyzer that checks if [Terraform](https://github.com/hashicorp/terraform/) files are correctly formatted | usable |


# SDK for Analyzer Developers

If you are developing an Analyzer, please check the [SDK documentation](sdk/README.md).

# Development

## Build

You can separately build the binaries provided by **lookout**; the binaries will be stored under `build/bin` directory.

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

To generate go code from [kallax](https://github.com/src-d/go-kallax) models, run:

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

For `lookout-sdk` integration tests (`-short` will skip tests that require bblfsh):
```bash
$ make test-sdk
$ make test-sdk-short
```

For `lookoutd serve` integration tests:
```bash
$ make test-json
```

## Dummy Analyzer Release

[Dummy analyzer](./cmd/dummy) is a simple analyzer implementation example. It is part of the lookout codebase but it's release cycle is managed independently from main one.

To release a new version and publish the dummy analyzer container you need to create a tag with the `dummy` prefix, e.g. `dummy-v0.0.1`. Please note this doesn't require to do a GitHub release, we just need the Git tag.

A normal release tag will not publish this container.


# Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](./CONTRIBUTING.md).

# Code of Conduct

All activities under source{d} projects are governed by the
[source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).

# License
Affero GPL v3.0, see [LICENSE](LICENSE).
