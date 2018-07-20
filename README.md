# lookout [![Build Status](https://travis-ci.org/src-d/lookout.svg)](https://travis-ci.org/src-d/lookout) ![GoDoc](https://godoc.org/gopkg.in/src-d/lookout?status.svg)](https://godoc.org/github.com/src-d/lookout)

A service for assisted code review, allows running custom code analysis on PRs.

# Installation

```
Short installation guide or link to longer installation instructions.
This should include a pre-requisites subsection if needed.

Are there Docker images, packages managers (brew, apt, etc), installations scripts?
```

The included [Docker Compose](https://docs.docker.com/compose/) file starts [bblfshd](https://github.com/bblfsh/bblfshd) and [PostgreSQL](https://www.postgresql.org/) containers.

* bblfsd listens on `localhost:9432`
* PostgreSQL listens on `localhost:5432`, with the superuser password `example`.

Clone the repository, or download `docker-compose.yml`, and run:

```bash
docker-compose up
```


# Example

## SDK

If you are developing an Analyzer, please check [SDK documentation](./sdk/README.md).
It includes curl-style binary that allows to trigger Analysers directly, without launchin a full lookout server.

## Server

To run a lookout server with defatult, dummy analyzer.

### Local

To trigger the analysis on actual PR of your Github repository do:

1. Start an analyzer
Any of the analysers, or a default dummy one, included in this repository
  ```
  go build -o analyzer ./cmd/dummy
  ./dummy
  ```
1. Start a lookout server
  `lookout serve <repository>`
1. Create a new PR in repository


### Docker

TBD

# Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look to
our [Contributing Guidelines](CONTRIBUTING.md).

# Code of Conduct

All activities under source{d} projects are governed by the [source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).

# License
AGPL v3.0, see [LICENSE](LICENSE).
