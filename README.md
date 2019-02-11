<a href="https://www.sourced.tech/lookout">
  <img src="./docs/assets/sourced-lookout.png" alt="source{d} Lookout" height="120px">
</a>

**Service for assisted code review, that allows running custom code Analyzers on pull requests.**

[![GitHub version](https://badge.fury.io/gh/src-d%2Flookout.svg)](https://github.com/src-d/lookout/releases)
[![Build Status](https://travis-ci.org/src-d/lookout.svg?branch=master)](https://travis-ci.org/src-d/lookout)
![Development](https://svg-badge.appspot.com/badge/stability/development?color=D6604A)
[![Code Coverage](https://codecov.io/github/src-d/lookout/coverage.svg)](https://codecov.io/github/src-d/lookout)
[![Go Report Card](https://goreportcard.com/badge/github.com/src-d/lookout)](https://goreportcard.com/report/github.com/src-d/lookout)
[![GoDoc](https://godoc.org/github.com/src-d/lookout?status.svg)](https://godoc.org/github.com/src-d/lookout)

[Website](https://www.sourced.tech) •
[Documentation](https://docs.sourced.tech) •
[Blog](https://blog.sourced.tech) •
[Slack](http://bit.ly/src-d-community) •
[Twitter](https://twitter.com/sourcedtech)

## Introduction

With **source{d} Lookout**, we’re introducing a service for assisted code review, that allows running custom code analyzers on pull requests.

Jump to the [Quickstart](#quickstart) section to start using it!


**Table of Contents**

<!--ts-->
* [Introduction](#introduction)
  * [Motivation and Scope](#motivation-and-scope)
  * [Current Status](#current-status)
  * [Further Reading](#further-reading)
* [Quickstart](#quickstart)
* [Available Analyzers](#available-analyzers)
* [Create an Analyzer](#create-an-analyzer)
* [Contribute](#contribute)
  * [Community](#community)
* [Code of Conduct](#code-of-conduct)
* [License](#license)
<!--te-->


### Motivation and Scope

source{d} is the company driving the Machine Learning on Code (#MLonCode) movement. Doing Machine Learning on Code consists of applying ML techniques to train models that can cluster, identify and predict useful aspects of source code and software repositories.

**source{d} Lookout** is the first step towards a full suite of Machine Learning on Code applications for AI-assisted coding, but you can also create your own analyzers without an ML approach.

The benefits of using **source{d} Lookout** are:
- Keep your code base style/patterns consistent.
- Language agnostic assisted code reviews.
- Identify where to focus your attention on code reviews.
- Automatically warn about common mistakes before human code review.

### Current Status

Currently, **source{d} Lookout** is in development process.

### Further Reading

This repository contains the code of **source{d} Lookout** and the project documentation, which you can also see properly rendered at [https://docs.sourced.tech/lookout](https://docs.sourced.tech/lookout).


## Quickstart

_There are several ways to run **source{d} Lookout**; we recommend to use `docker-compose` because it's straightforward, but you can [learn more about the **different ways to run source{d} Lookout**](/docs/how-to-run.md)._

_Please refer to the [**Configuring source{d} Lookout** guide](/docs/configuration.md) for documentation about the `config.yml` file, and to know how to configure **source{d} Lookout** to analyze your repositories, or to use your own analyzers._

There is [`docker-compose.yml`](/docker-compose.yml) config file for [Docker Compose](https://docs.docker.com/compose) to start **source{d} Lookout**, its dependencies (**bblfsh** and **PostgreSQL**) and a [`dummy` analyzer](analyzers-examples.md#dummy-analyzer) which will add some stats to the watched pull requests.

To do so, clone this repository or download [`docker-compose.yml`](/docker-compose.yml) directly.

Create the `config.yml` file in the same directory where `docker-compose.yml` is. You can use [`config.yml.tpl`](/config.yml.tpl) as a template. Make sure that you specify in the `config.yml` the repositories that will be watched by **source{d} Lookout**. Then run, passing [a valid GitHub user/token](docs/configuration.md#authentication-with-github):

```bash
$ docker-compose pull
$ GITHUB_USER=<user> GITHUB_TOKEN=<token> docker-compose up --force-recreate
```

Once it is running, **source{d} Lookout** will start posting the comments returned by `dummy` analyzer into the pull requests opened at GitHub in the repositories that you configured to be watched.

You can stop it by pressing `ctrl+c`.

If you want to try **source{d} Lookout** with your own analyzer instead of `dummy` one, you must run it in advance, then [set it into `config.yml`](/docs/configuration.md#analyzers) and then run:

```bash
$ docker-compose pull
$ GITHUB_USER=<user> GITHUB_TOKEN=<token> docker-compose up --force-recreate lookout bblfsh postgres
```

If you need to reset the database to a clean state, you should drop the `postgres` container. To do so, stop running **source{d} Lookout** with `ctrl+c` and then execute:

```bash
$ docker rm lookout_postgres_1
```


## Available Analyzers

This is the list of the known implemented analyzers for **source{d} Lookout**:

| Name | Description | Targeted files | Maturity level |
| --- | --- | --- | --- |
| [style-analyzer](https://github.com/src-d/style-analyzer) | Code style analyzer |  | development |
| [terraform](https://github.com/src-d/lookout-terraform-analyzer) | Checks if [Terraform](https://github.com/hashicorp/terraform/) files are correctly formatted | Terraform | usable |
| [gometalint](https://github.com/src-d/lookout-gometalint-analyzer) | Reports [gometalinter](https://github.com/alecthomas/gometalinter) results on pull requests | Go | testing and demo |
| [sonarcheck](https://github.com/src-d/lookout-sonarcheck-analyzer) | Reports [SonarSource](https://github.com/bblfsh/sonar-checks) checks results on pull requests using [bblfsh UAST](https://doc.bblf.sh/uast/uast-specification.html) | Java | testing and demo |
| [flake8](https://github.com/src-d/lookout-flake8-analyzer) | Reports [flake8](http://flake8.pycqa.org/en/latest/) results on pull requests | Python| testing and demo |
| [npm-audit](https://github.com/erizocosmico/npm-audit-analyzer) | Reports issues with newly added dependencies using [npm-audit](https://docs.npmjs.com/cli/audit) | JavaScript | development |
| [function-name analyzer](https://github.com/src-d/function-name-analyzer) | Applies a translation model from function identifiers to function names. |  | development |


## Create an Analyzer

If you are developing an Analyzer, or you want more info about how they work, please check the [documentation about **source{d} Lookout** analyzers](/docs/analyzers.md).


## Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](/docs/CONTRIBUTING.md).

### Community

source{d} has an amazing community of developers and contributors who are interested in Code As Data and/or Machine Learning on Code. Please join us! 👋

- [Slack](http://bit.ly/src-d-community)
- [Twitter](https://twitter.com/sourcedtech)
- [Email](mailto:hello@sourced.tech)


## Code of Conduct

All activities under source{d} projects are governed by the
[source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).


## License

Affero GPL v3.0 or later, see [LICENSE](LICENSE.md).
