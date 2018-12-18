# Running source{d} Lookout

The recommended way to locally try **source{d} Lookout** is using `docker-compose` as described in [quickstart documentation](../README.md#quickstart).

In some situations, you might want to run all its dependencies and components separately, or you may even want to [run it in a distributed mode](#running-lookout-in-distributed-mode). Both ways to run **source{d} Lookout** would require to follow these general steps:

1. [Run source{d} Lookout dependencies](#dependencies).
1. [Run the desired analyzers](#run-the-analyzers).
1. [Configure source{d} Lookout](#configure-lookout).
1. [Run source{d} Lookout](#run-lookout).

In other situations you may just want to test an analyzer locally without accessing GitHub at all. For those situations, you might want to read the [`lookout-sdk` binary documentation](lookout-sdk.md).

## Running source{d} Lookout in Distributed Mode

**source{d} Lookout** can be also run in a distributed fashion using a [RabbitMQ](https://www.rabbitmq.com/) queue to coordinate a _watcher_ and several _workers_.

- The _watcher_ process monitors GitHub pull requests and enqueues jobs for new events.
- The running _workers_ dequeue jobs as they become available, call the registered analyzers, and post the results as comments.

The general steps to run source{d} Lookout in distributed mode are the same as said above.


# Dependencies

**source{d} Lookout** needs a running instance of:

* [bblfshd](https://github.com/bblfsh/bblfshd) to parse files into [UAST](https://doc.bblf.sh/uast/uast-specification.html).
* [PostgreSQL](https://www.postgresql.org) for persistence.
* (**optional**) [RabbitMQ](https://www.rabbitmq.com) to coordinate a _watcher_ and several _workers_ (when running **source{d} Lookout** in a distributed way).

You can run them manually or with `docker-compose`.

```bash
$ docker-compose up -d --force-recreate bblfsh postgres
```

In case you want to run it in a distributed way, you will also need RabbitMQ, so you can run instead:

```bash
$ docker-compose -f docker-compose.yaml -f docker-compose-rabbitmq.yml up -d --force-recreate bblfsh postgres rabbitmq
```

To monitor RabbitMQ, go to http://localhost:8081, and access it with `guest/guest`

# Run the Analyzers

You will need to run the Analyzers to be used by **source{d} Lookout**.

You can run one of our [example analyzers](analyzers-examples.md), any of the already [available analyzers](../README.md#available-analyzers) or the one that you're developing.

For testing purposes, you may want to use a [`dummy` analyzer](analyzers-examples.md#dummy-analyzer). You can download it from [**source{d} Lookout** releases page](https://github.com/src-d/lookout/releases) and then run it:

```bash
$ dummy serve
```

# Configure source{d} Lookout

Copy the [`config.yml.tpl`](/config.yml.tpl) into `config.yml` and modify it according to your needs.

Take a look at [configuration and GitHub authentication](configuration.md) for more details about **source{d} Lookout** configuration.

At least you should:

1. Add the gRPC addresses of the analyzers you ran in the previous step.
1. Add the URLs of the repositories to be watched or authenticate as a GitHub App.


# Run source{d} Lookout

Download the latest `lookoutd` binary from [**source{d} Lookout** releases page](https://github.com/src-d/lookout/releases).

## Initialize the Database

_For non-default configuration, please take a look into [**`lookoutd` Command Options**](#options)_

```bash
$ lookoutd migrate
```

## Start source{d} Lookout

_For non-default configuration, please take a look into [**`lookoutd` Command Options**](#options)_

For a single server watching GitHub and processing events, just run:

```bash
$ lookoutd serve [--dry-run] [--github-token=<token> --github-user=<user>]
```

### Distributed Mode

_For non-default configuration, please take a look into [**`lookoutd` Command Options**](#options)_

In order to run it in a distributed mode, the _watcher_ and the _workers_ must be run separately.

Run the _watcher_:

```bash
$ lookoutd watch [--github-token=<token> --github-user=<user>]
```

and as many _workers_ you need:

```bash
$ lookoutd work [--dry-run] [--github-token=<token> --github-user=<user>]
```

<a id=options></a>
# Appendix: `lookoutd` Command Options

`lookoutd` binary includes some subcommands as described above, and they accept many different options; you can use:
- `lookoutd -h`, to see all the available subcommands.
- `lookoutd subcommand -h`, to see all the options for the given subcommand.

Here are some of the most relevant options for `lookoutd`:

- [dry-run mode](#dry-run-mode)
- [authentication options](#authentication-options)
- [number of concurrent events to process](#concurrent-events)
- [dependencies URIs](#dependencies-uris)
- [logging options](#logging-options)

## Dry-run Mode
If you want to avoid posting the analysis results on GitHub, and only print them, enable the _dry-run_ mode when running `serve`, `work` subcommands:

| subcommands | Env var | Option |
| --- | --- | --- |
| `serve`, `work` | `LOOKOUT_DRY_RUN`  | `--dry-run` |

## Authentication Options

To post the comments returned by the Analyzers into GitHub, you can configure the authentication in the `config.yml` (see [configuration documentation](configuration.md)), or do it explicitly when running `serve`, `work` and `watch` subcommands:

| subcommands | Env var | Option |
| --- | --- | --- |
| `serve`, `work`, `watch` | `GITHUB_USER`  | `--github-user=` |
| `serve`, `work`, `watch` | `GITHUB_TOKEN`  | `--github-token=` |

<a id=concurrent-events></a>
## Number of Concurrent Events to Process

You can adjust the number of events that each _worker_ or the single _server_ will process concurrently when running `serve` or `work` subcommands (if you set it to `0`, it will process as many as the number of processors you have):

| subcommands | Env var | Option | Default |
| --- | --- | --- | --- |
| `serve`, `work` | `LOOKOUT_WORKERS`  | `--workers=` | 1 |

## Dependencies URIs

If you started all the **source{d} Lookout** dependencies using `docker-compose`, then `lookoutd` binary will be able to find them with its default values; otherwise, you should pass some extra values when running the `lookoutd` binary:

| subcommands | Env var | Option | Description | Default |
| --- | --- | --- | --- | --- |
| `serve`, `work`, `migrate` | `LOOKOUT_DB`  | `--db=`  | **PostgreSQL** connection string | `postgres://postgres:postgres@localhost:5432/lookout?sslmode=disable` |
| `serve`, `work` | `LOOKOUT_BBLFSHD`  | `--bblfshd=`  | **bblfsh** gRPC address | `ipv4://localhost:9432` |
| `watch`, `work` | `LOOKOUT_QUEUE`  | `--queue=`  | **RabbitMQ** queue name | `lookout` |
| `watch`, `work` | `LOOKOUT_BROKER`  | `-broker-=`  | **RabbitMQ** broker service URI | `amqp://localhost:5672` |

## Logging options

| Env var | Option | Description | Default |
| --- | --- | --- | --- |
| `LOG_LEVEL` | `--log-level=` | Logging level (`info`, `debug`, `warning` or `error`) | `info` |
| `LOG_FORMAT`| `--log-format=` | log format (`text` or `json`), defaults to `text` on a terminal and `json` otherwise | |
| `LOG_FIELDS` | `--log-fields=` | default fields for the logger, specified in json | |
| `LOG_FORCE_FORMAT` | `--log-force-format` | ignore if it is running on a terminal or not | |
