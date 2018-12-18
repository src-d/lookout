# lookout-sdk Binary

_For the **lookout-sdk** library to develop new analyzers go to [**lookout-sdk**](https://github.com/src-d/lookout-sdk) repository._

`lookout-sdk` binary is a simplified version of the **Lookout** server that works with a local git repository and does not need access to Github.

You can think about `lookout-sdk` as a _curl-like_ tool to call an analyzer gRPC endpoint with a `ReviewEvent` or a `PushEvent`, from a local git repository, and send it to an analyzer without accessing GitHub at all. For convenience, `lookout-sdk` also exposes a **Lookout DataService** backed by the same git repository.

You can download the latest `lookout-sdk` from the [src-d/lookout releases page](https://github.com/src-d/lookout/releases).

This is the sequence diagram of the `ReviewEvent` made by `lookout-sdk review`. You can compare it with a regular **Lookout** installation at the [**source{d} Lookout architecture** documentation](architecture.md).

![sequence diagram](assets/lookout-sdk-seq-diagram.png)

You can [edit this image](https://mermaidjs.github.io/mermaid-live-editor/#/edit/eyJjb2RlIjoic2VxdWVuY2VEaWFncmFtXG4gICAgcGFydGljaXBhbnQgc2RrIGFzIGxvb2tvdXQtc2RrXG4gICAgcGFydGljaXBhbnQgQW5hbHl6ZXJcbiAgICBwYXJ0aWNpcGFudCBCYWJlbGZpc2hcbiAgICBzZGstPj5BbmFseXplcjogTm90aWZ5UmV2aWV3RXZlbnRcbiAgICBBbmFseXplci0-PnNkazogR2V0Q2hhbmdlcy9HZXRGaWxlc1xuICAgIHNkay0-PkJhYmVsZmlzaDogcGFyc2VGaWxlIChpZiBXYW50VUFTVClcbiAgICBCYWJlbGZpc2gtLT4-c2RrOiBVQVNUXG4gICAgc2RrLS0-PkFuYWx5emVyOiBDaGFuZ2UvRmlsZVxuICAgIEFuYWx5emVyLS0-PnNkazogQ29tbWVudHNcbiIsIm1lcm1haWQiOnsidGhlbWUiOiJkZWZhdWx0In19) using [mermaid](https://mermaidjs.github.io). ([sourcecode](assets/lookout-sdk-seq-diagram.md))

## Requirements

You will need to run an analyzer to be called by `lookout-sdk`. You can run one of our [example analyzers](analyzers-examples.md), any of the already [available analyzers](../README.md#available-analyzers) or the one that you're developing.

If your analyzer makes use of UAST, you will also need a [Babelfish server](https://doc.bblf.sh/using-babelfish/getting-started.html) running.
To start it using [Docker Compose](https://docs.docker.com/compose/) clone this repository, or download [`docker-compose.yml`](../docker-compose.yml), and run:

```bash
$ docker-compose up bblfshd
```

This will create the [bblfshd](https://github.com/bblfsh/bblfshd) container listening on `localhost:9432`.


## Usage

To perform a `NotifyReviewEvent` call to an analyzer and serve the **Lookout DataService** endpoint, run:
```shell
$ lookout-sdk review
```

To perform a `NotifyPushEvent` call to an analyzer and serve the **Lookout DataService** endpoint, run:
```shell
$ lookout-sdk push
```

In the next section, you will find a more detailed example considering the most usual options for running `lookout-sdk` against any analyzer from two given revisions.


## How Does It Work

If we look at this example history of a given local repository stored at `/somewhere/repo/path`:

```bash
$ git log --pretty=oneline --graph

*   d036524c463227524f4bbd7b207fb87bb8b89ee3 (HEAD -> master) Merge pull request #3
|\
| * 045a24828327ac35a28186f9b9b437adc3f7b7a3 (branch-b) message
| * 804cbd94869cb173494ce1de410f2b48674bc772 message
|/
*   9294ddb13cc7c8acd2db480c9e5c1396cd85e50a (branch-a) message
*   7f2ee64cd0a5891900cc368ae35e60a61c262060 message
*
...
```

And considering we have an analyzer listening at `ipv4://localhost:9999`

we could run:

```shell
$ lookout-sdk review \
  --git-dir=/somewhere/repo/path \
  --from=branch-a \
  --to=branch-b \
  "ipv4://localhost:9999"
```

Doing so, `lookout-sdk` will:

1. start a gRPC **Lookout DataService** endpoint backed by the repository stored at `/somewhere/repo/path`;
1. create a `ReviewEvent` with the changes between `branch-b` and `branch-a`;
1. send a gRPC `NotifyReviewEvent` call to your analyzer listening on `ipv4://localhost:9999` with the created `ReviewEvent`;
1. wait till the analyzer sends a response with the comments; the analyzer will be able to request file contents, file language or UASTs to the gRPC **Lookout DataService** endpoint exposed by `lookout-sdk`;
1. once the analyzer sends the response, `lookout-sdk` will put it into the `STDOUT`, stop the **Lookout DataService** and exit.

_For more options to run `lookout-sdk`, take a look into [**lookout-sdk Command Options**](#options)_

- If analyzer gRPC address is omitted, it will be `ipv4://localhost:9930`.
- If `--git-dir` is omitted, the current dir will be used.
- If `--from` is omitted, it will be `HEAD^`.
- If `--to` is omitted, it will be `HEAD`.
- Both `--from` and `--to` can be any [git revision](https://git-scm.com/docs/gitrevisions#_specifying_revisions). For example a tag name, branch name or the full commit SHA-1.

Everything explained above for `lookout-sdk review` calling `NotifyReviewEvent`, applies also to `NotifyPushEvent` when using `lookout-sdk push`.


<a id=options></a>
# Appendix: `lookout-sdk` Command Options

`lookout-sdk` binary include some subcommands as described above, and they accept many different options; you can use:
- `lookout-sdk -h`, to see all the available subcommands.
- `lookout-sdk subcommand -h`, to see all the options for the given subcommand.

Here are some of the most relevant options for both `lookout-sdk push` and `lookout-sdk review`:

| Env var | Option | Description | Default |
| --- | --- | --- | --- |
| `LOOKOUT_BBLFSHD` | `--bblfshd=` | gRPC URL of the Bblfshd server | `ipv4://localhost:9432` |
| `GIT_DIR` | `--git-dir=` | path to the Git directory to analyze | `.` _(current dir)_ |
| | `--from=` | name of the base [git revision](https://git-scm.com/docs/gitrevisions#_specifying_revisions) for event | `HEAD^` |
| | `--to=` | name of the head [git revision](https://git-scm.com/docs/gitrevisions#_specifying_revisions) for event | `HEAD` |
| | `--config-json=` | arbitrary JSON configuration for request to an analyzer | |

## Logging Options

| Env var | Option | Description | Default |
| --- | --- | --- | --- |
| `LOG_LEVEL` | `--log-level=` | Logging level (`info`, `debug`, `warning` or `error`) | `info` |
| `LOG_FORMAT`| `--log-format=` | log format (`text` or `json`), defaults to `text` on a terminal and `json` otherwise | |
| `LOG_FIELDS` | `--log-fields=` | default fields for the logger, specified in json | |
| `LOG_FORCE_FORMAT` | `--log-force-format` | ignore if it is running on a terminal or not | |
