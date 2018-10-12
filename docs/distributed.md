# Running in Distributed Mode

lookout can be run in a distributed fashion using a [RabbitMQ](https://www.rabbitmq.com/) queue to coordinate a _watcher_ and several _workers_.

The _watcher_ process monitors GitHub pull requests and enqueues jobs for new events. One (or more) _workers_ dequeue jobs as they become available, analyze the pull requests with the help of bblfsh and the analyzers, and post the results as comments.

This deployment still uses the same configuration file as a regular deployment, see the [**Configuring lookout guide**](docs/configuration.md) for more information.

To get started, follow these steps:

- Run the dependencies:
```bash
docker-compose up -d --force-recreate postgres bblfsh
```
- Start RabbitMQ. To monitor it, go to http://localhost:8081, and access with `guest/guest`:
```bash
docker run -d --hostname rabbitmq --name rabbitmq -p 8081:15672 -p 5672:5672 rabbitmq:3-management
```
- Download the `lookoutd` binary from [the releases page](https://github.com/src-d/lookout/releases), and run:
```bash
lookoutd watch

lookoutd work
```

The `lookoutd watch` and `work` subcommands accept the following options to configure the queue options:

| Env var | Option | Description | Default |
| -- | -- | -- | -- |
| `LOOKOUT_QUEUE`  | `--queue=`  | queue name | `lookout` |
| `LOOKOUT_BROKER` | `--broker=` | broker service URI | `amqp://localhost:5672` |

You can also adjust the number of events that each _worker_ will process concurrently:

| Env var | Option | Description | Default |
| -- | -- | -- | -- |
| `LOOKOUT_WORKERS` | `--workers=` | number of concurrent workers processing events, 0 means the same number as processors | 1  |

