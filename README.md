# lookout [![Build Status](https://travis-ci.org/src-d/lookout.svg)](https://travis-ci.org/src-d/lookout) [![GoDoc](https://godoc.org/gopkg.in/src-d/lookout?status.svg)](https://godoc.org/github.com/src-d/lookout)

A service for assisted code review, that allows running custom code Analyzers on pull requests.

# SDK

If you are developing an Analyzer, please check [SDK documentation](./sdk/README.md).

It includes a curl-style binary `lookout-sdk` that allows to trigger Analyzers directly, without launching a full lookout server.

# Installation

`go get github.com/src-d/lookout`

# Dependencies

The included [`./docker-compose.yml`](./docker-compose.yml) allows to start all dependencies using [Docker Compose](https://docs.docker.com/compose/) 

* [bblfshd](https://github.com/bblfsh/bblfshd), on `localhost:9432`
* [PostgreSQL](https://www.postgresql.org/), on `localhost:5432` password `postgres`

Clone the repository, or download [`./docker-compose.yml`](./docker-compose.yml)

# Usage

To trigger the analysis on an actual pull request of a GitHub repository you will need [GitHub access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/).

## With Docker

Run:

```bash
GITHUB_USER=<user> GITHUB_TOKEN=<token> REPO=github.com/<user>/<name> docker-compose up
```

## Without Docker

1. Run dependencies manually or using docker-compose:
    ```bash
    docker-compose up bblfsh postgres
    ```
1. Initialize the database. This command will work for the PostgreSQL created by docker-compose, use `-h` to see other options.
    ```bash
    lookoutd migrate
    ```
1. Start an analyzer
Any of the analyzers or a default dummy one, included in this repository
    ```
    go build -o analyzer ./cmd/dummy
    ./analyzer serve
    ```
1. Start a lookout server
    1. With posting analysis results on GitHub
        - Obtain [GitHub access token](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
        - Run `lookoutd serve --github-token <token> --github-user <user> <repository>`
    1. Without posting analysis results (only printing)
        - `lookoutd serve --dry-run <repository>`


# Configuration file

Global server configuration is stored in `config.yml`:

```yml
analyzers:
  - name: Example name # required, unique name of the analyzer
    addr: ipv4://localhost:10302 # required, gRPC address
    disabled: false # optional, false by default
    settings: # optional, this field is sent to analyzer "as is"
        threshold: 0.8
```

It's possible to override Analyzers configuration for a particular repository.
To do that `.lookout.yml` must be present in the root of that repository.

Example:
```yml
analyzers:
  - name: Example name # must be the same as in server configuration, unknown names will be ignored
    disabled: true # local configuration can only disable analyzer, not enable
    settings: # settings for an analyzer will be merged with a global one
        threshold: 0.9
        mode: confident
```

Merging rules:
- Objects are deep merged
- Arrays are replaced
- Null value replaces object

# Authenticate as a GitHub App

Instead of using a GitHub username and token you can use lookout as a [GitHub App](https://developer.github.com/apps/about-apps/).

You need to create a new GitHub App following the [GitHub documentation](https://developer.github.com/apps/building-github-apps/creating-a-github-app/). Then download a private key ([see how here](https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/)) and set the following fields in your `config.yml` file:

```yml
providers:
  github:
    app_id: 1234
    private_key: ./key.pem
```

You should also unset any environment variable or option for the GitHub username and token authentication.

_Note_: This authentication method is still under development. There are some caveats you should be aware of:

When using this authentication method the repositories to analyze are retrieved from the GitHub installations.
This means that the positional argument for `lookoutd serve` is ignored. You should also be aware that the list of repositories is retrieved only once when the server starts.

# Contribute

[Contributions](https://github.com/src-d/lookout/issues) are more than welcome, if you are interested please take a look to
our [Contributing Guidelines](CONTRIBUTING.md).

# Code of Conduct

All activities under source{d} projects are governed by the [source{d} code of conduct](https://github.com/src-d/guide/blob/master/.github/CODE_OF_CONDUCT.md).

# License
Affero GPL v3.0, see [LICENSE](LICENSE).

SDK package in `./sdk` is released under the terms of the [Apache License v2.0](./sdk/LICENSE)
