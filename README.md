<h1 align="center">
  <br>
  <a href="https://www.sourced.tech"><img src="./docs/assets/sourced.png" alt="source{d}" height="60px"></a>
  <br>
  <br>
  source{d} Lookout
  <br>
</h1>

<h3 align="center">
  Service for assisted code review, that allows running custom code Analyzers on pull requests.
</h3>

<p align="center">
  <a href="https://github.com/src-d/lookout/releases">
    <img src="https://badge.fury.io/gh/src-d%2Flookout.svg"
         alt="GitHub version">
  </a>
  <a href="https://travis-ci.org/src-d/lookout">
    <img src="https://travis-ci.org/src-d/lookout.svg?branch=master"
         alt="Build Status">
  </a>
  <img src="https://svg-badge.appspot.com/badge/stability/development?color=D6604A"
         alt="Development">
  <a href="https://codecov.io/github/src-d/lookout">
    <img src="https://codecov.io/github/src-d/lookout/coverage.svg"
         alt="Code Coverage">
  </a>
  <a href="https://goreportcard.com/report/github.com/src-d/lookout">
    <img src="https://goreportcard.com/badge/github.com/src-d/lookout"
         alt="Go Report Card">
  </a>
  <a href="https://godoc.org/github.com/src-d/lookout">
    <img src="https://godoc.org/github.com/src-d/lookout?status.svg"
         alt="GoDoc">
  </a>
  <a href="https://docs.google.com/document/d/1pqz-_SHO5BsJE-aa8o_bAY3r5vR67amnWN8-qZc2UgY">
    <img src="https://img.shields.io/badge/source%7Bd%7D-design%20document-blue.svg"
         alt="source{d} design document">
  </a>
</p>

<p align="center"><b>
    <a href="https://www.sourced.tech">Website</a> •
    <a href="https://docs.sourced.tech">Documentation</a> •
    <a href="https://blog.sourced.tech">Blog</a> •
    <a href="http://bit.ly/src-d-community">Slack</a> •
    <a href="https://twitter.com/sourcedtech">Twitter</a>
</b></p>


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

<!-- Added by: david, at: 2018-12-07T15:16+01:00 -->

<!--te-->


# Introduction

With source{d} Lookout, we’re introducing a service for assisted code review, that allows running custom code analyzers on pull requests.

Jump to the [Quickstart](#quickstart) section to start using it!

## Motivation and Scope

source{d} is the company driving the Machine Learning on Code (#MLonCode) movement. Doing Machine Learning on Code consists of applying ML techniques to train models that can cluster, identify and predict useful aspects of source code and software repositories.

source{d} Lookout is the first step towards a full suite of Machine Learning on Code applications for AI-assisted coding, but you can also create your own analyzers without an ML approach.

The benefits of using source{d} lookout are:
- Keep your code base style/patterns consistent,
- Language Agnostic assisted code reviews,
- Identify where to focus your attention on code reviews,
- Automatically warn about common mistakes before human code review.

## Current Status

Currently, source{d} Lookout is in development process.

## Further Reading

This repository contains the code of `lookout` and the project documentation, which you can also see properly rendered at [https://docs.sourced.tech/lookout](https://docs.sourced.tech/lookout).


# Quickstart

_There are different ways to run lookout; we recommend to use `docker-compose` because it's straightforward, but you can [learn more about the **different ways to run lookout**](/docs/how-to-run.md)._

_Please refer to the [**Configuring source{d} lookout** guide](/docs/configuration.md) for documentation about the `config.yml` file._

Using [Docker Compose](https://docs.docker.com/compose) you can use the provided [`docker-compose.yml`](/docker-compose.yml) config file to start **lookout**, its dependencies (**bblfsh** and **PostgreSQL**) and a `dummy` analyzer which will add some stats to the watched pull requests.

To do so, clone this repository or download [`docker-compose.yml`](/docker-compose.yml).

Create the `config.yml` file in the same directory where `docker-compose.yml` is (you can use [`config.yml.tpl`](/config.yml.tpl) as a template), and then run:

```bash
$ docker-compose pull
$ GITHUB_USER=<user> GITHUB_TOKEN=<token> docker-compose up --force-recreate
```

If you need to restart the database to a clean state, you can do so with:
```bash
$ docker rm lookout_postgres_1
```


# Available Analyzers

This is a list of some of the available analyzers for lookout:

| Name | Description | Targeted files | Maturity level |
| -- | -- | -- |-- |
| [style-analyzer](https://github.com/src-d/style-analyzer) | Code style analyzer |  | development |
| [terraform](https://github.com/src-d/lookout-terraform-analyzer) | Checks if [Terraform](https://github.com/hashicorp/terraform/) files are correctly formatted | Terraform | usable |
| [gometalint](https://github.com/src-d/lookout-gometalint-analyzer) | Reports [gometalinter](https://github.com/alecthomas/gometalinter) results on pull requests | Go | testing and demo |
| [sonarcheck](https://github.com/src-d/lookout-sonarcheck-analyzer) | Reports [SonarSource](https://github.com/bblfsh/sonar-checks) checks results on pull requests using [bblfsh UAST](https://doc.bblf.sh/uast/uast-specification.html) | Java | testing and demo |
| [flake8](https://github.com/src-d/lookout-flake8-analyzer) | Reports [flake8](http://flake8.pycqa.org/en/latest/) results on pull requests | Python| testing and demo |
| [npm-audit](https://github.com/erizocosmico/npm-audit-analyzer) | Reports issues with newly added dependencies using [npm-audit](https://docs.npmjs.com/cli/audit) | JavaScript | development |
| [function-name analyzer](https://github.com/src-d/function-name-analyzer) | Applies a translation model from function identifiers to function names. |  | development |


# Create an Analyzer

If you are developing an Analyzer, or you want more info about how do they work, please check the [documentation about lookout analyzers](/docs/analyzers.md).


# Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](/docs/CONTRIBUTING.md).

## Community

source{d} has an amazing community of developers and contributors who are interested in Code As Data and/or Machine Learning on Code. Please join us! 👋

- [Slack](http://bit.ly/src-d-community)
- [Twitter](https://twitter.com/sourcedtech)
- [Email](mailto:hello@sourced.tech)


# Code of Conduct

All activities under source{d} projects are governed by the
[source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).


# License
Affero GPL v3.0 or later, see [LICENSE](LICENSE.md).
