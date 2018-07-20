# Lookout Analyzer SDK

An analyzer is a gRPC server, that implements [Analyzer service](https://github.com/src-d/lookout/tree/master/sdk/service_analyzer.proto) to receive events from the server.

An analyzer should use gRPC client to access [Data service](https://github.com/src-d/lookout/tree/master/sdk/service_data.proto) to get content or UAST of the changes.

All `.proto` files are located in [/sdk](https://github.com/src-d/lookout/tree/master/sdk) directory.


## Code generation

### Python

Dependencies:

```bash
$ pip install grpcio-tools
```

Read more about gRPC in [Python Quickstart](https://grpc.io/docs/quickstart/python.html).

Generation:

```bash
$ PY_OUT_DIR=<directory for generated files> mkdir -p $PY_OUT_DIR && \
    python -m grpc_tools.protoc -Isdk \
    --python_out=$PY_OUT_DIR --grpc_python_out=$PY_OUT_DIR \
    sdk/*.proto
```

## Testing

Download the latest lookout binary from [releases](https://github.com/src-d/lookout/releases) page.

Babelfish server is required. If you don't have it running, please read the [getting started guide](https://doc.bblf.sh/using-babelfish/getting-started.html), to learn more about how to start and use it.

Run the binary inside a git repository directory with gRPC address of the analyzer

```bash
$ ./lookout review ipv4://localhost:10302
```

By default it would trigger review event with changes from `HEAD^` to `HEAD`.
You can change it using additional flags `--from` and `--to`. Both flags accept [git revision](https://git-scm.com/docs/gitrevisions#_specifying_revisions).

# License
Apache 2.0, see [LICENSE](LICENSE).