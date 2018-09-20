lookout
[![Build Status](https://travis-ci.org/src-d/lookout.svg)](https://travis-ci.org/src-d/lookout)
[![GoDoc](https://godoc.org/gopkg.in/src-d/lookout?status.svg)](https://godoc.org/github.com/src-d/lookout)
[![Code Coverage](https://codecov.io/github/src-d/lookout/coverage.svg)](https://codecov.io/github/src-d/lookout)
[![Go Report Card](https://goreportcard.com/badge/github.com/src-d/lookout)](https://goreportcard.com/report/github.com/src-d/lookout)
![development](https://svg-badge.appspot.com/badge/stability/development?color=D6604A)
=======

A service for assisted code review, that allows running custom code Analyzers on pull requests.


# Installation

```bash
$ go get github.com/src-d/lookout/...
```

It will globally install `lookoutd`, `lookout-sdk`, and `dummy`.


## Dependencies

**lookout** will run custom Analyzers to perform the analysis of the PR.
It also needs a running instance of:

* [bblfshd](https://github.com/bblfsh/bblfshd) to parse files into [UAST](https://doc.bblf.sh/uast/uast-specification.html)
* [PostgreSQL](https://www.postgresql.org) to store the analysis events and logs


# Usage

The **lookout** configuration is defined by the `config.yml` file; you can use the template [`config.yml.tpl`](config.yml.tpl) to create your own config file. You will find more information about it in the [docs about how to configure lookout](docs/configuration.md)


To trigger the analysis on any pull request of a GitHub repository you will need a GitHub authentication as it is described in the [docs about how to authenticate with GitHub](docs/configuration.md#basic-auth)


## With docker-compose

Using [Docker Compose](https://docs.docker.com/compose) you can use the provided [`docker-compose.yml`](docker-compose.yml) config file to start **lookout**, its dependencies (**bblfsh** and **PostgreSQL**) and the `dummy` analyzer which will add some stats to the watched pull requests.

To do so, just clone this repository or download [`docker-compose.yml`](docker-compose.yml).

Create the `config.yml` file in the same directory than the `docker-compose.yml` one, and run from there:

```bash
$ GITHUB_USER=<user> GITHUB_TOKEN=<token> REPO=github.com/<user>/<name> docker-compose up
```

## Running lookout from binaries

Following these, steps you will be able to run separately the **lookout** dependencies, analyzers and the server itself.

1. Run the [dependencies](#dependencies) manually or using docker-compose; if you prefer doing it straightforward, just run:

    ```bash
    $ docker-compose up bblfsh postgres
    ```

1. Initialize the database.<br />
    The following command will work for the PostgreSQL created by `docker-compose` as explained above; otherwise, you might need to define the use connection string to PostgreSQL database; use `-h` to see other options.

    ```bash
    $ lookoutd migrate
    ```

1. Start an analyzer before running **lookout**.<br />
    You can use the *dummy* one as it is provided by this repository; to do so just run:

    ```bash
    $ dummy serve
    ```

1. Copy the [`config.yml.tpl`](config.yml.tpl) into `config.yml` and add the URLs of the repositories to be watched. (take a look into [configuration and GitHub authentication](docs/configuration.md) for more details about **lookout** configuration)

1. Start **lookout** server<br />
    If you want to post the analysis results on GitHub, run:

    ```bash
    $ lookoutd serve --github-token <token> --github-user <user>
    ```

    If you want to avoid posting the analysis results on GitHub, and only print them, run:

    ```bash
    $ lookoutd serve --dry-run
    ```


# Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](docs/CONTRIBUTING.md).

You have more information on how to run it locally for [development purposes here](docs/CONTRIBUTING.md#development).

If you are developing an Analyzer, please check [SDK documentation](sdk/README.md).


# Code of Conduct

All activities under source{d} projects are governed by the
[source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).


# License
Affero GPL v3.0, see [LICENSE](LICENSE).
