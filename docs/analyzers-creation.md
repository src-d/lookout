# Implementing Your Own Analyzer

_For a brief description about what is an analyzer, you can read [**source{d} Lookout Analyzers** documentation](analyzers.md)_

_Please refer to the [**official Protocol Buffers** documentation](https://developers.google.com/protocol-buffers/) to learn how to get started with Protocol Buffers._

To implement your own analyzer you only need to create a gRPC service implementing the [Analyzer service](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_analyzer.proto#L30) interface:

```protobuf
service Analyzer {
  rpc NotifyReviewEvent (ReviewEvent) returns (EventResponse);
  rpc NotifyPushEvent (PushEvent) returns (EventResponse);
}
```

You can create a new analyzer in any language that supports protocol buffers, generating code from [the `.proto` definitions](https://github.com/src-d/lookout-sdk/tree/master/proto/lookout/sdk). The resulting code will provide data access classes, with accessors for each field, as well as methods to serialize/parse the message structures to/from bytes.

## Caveats

All the analyzers should consider [the caveats described by the SDK](https://github.com/src-d/lookout-sdk#caveats).


## Fetching Changes, UASTs or Languages from DataService

**source{d} Lookout** will take care of dealing with Git repositories, UAST extraction, programming language detection, etc. Your analyzer will be able to use the **DataService** to query all this data.

You can read more about it in the [**source{d} Lookout Server** section](architecture.md#server).


## How to Test an Analyzer Locally

_Please refer to [**lookout-sdk** docs](lookout-sdk.md) to see how to locally test an analyzer without accessing GitHub at all._


# Using Pregenerated Code from the SDK

If you're creating your analyzer in Golang or Python, you'll find pre-generated libraries in the [lookout-sdk repository](https://github.com/src-d/lookout-sdk). The SDK libraries also come with helpers to deal with gRPC caveats.

**lookout-sdk** repository contains a [quickstart example](https://github.com/src-d/lookout-sdk/blob/master/examples) &mdash;implemented in Go and in Python&mdash; of an Analyzer that detects the language and number of functions for every file.

You can do as it follows:

## Golang

Import and use `gopkg.in/src-d/lookout-sdk.v0/pb`.

The analyzer must implement the [AnalyzerServer interface](https://godoc.org/gopkg.in/src-d/lookout-sdk.v0/pb#AnalyzerServer):

Once you register the analyzer and the gRPC server is runing, it will listen for requests from `lookoutd`.

_example:_

```go
package main
import (
	"context"

	"google.golang.org/grpc"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

type analyzer struct{}

func (*analyzer) NotifyReviewEvent(ctx context.Context, review *pb.ReviewEvent) (*pb.EventResponse, error) {
	var comments []*pb.Comment

	// Logic to generate 'comments' given the passed 'review'

	return &pb.EventResponse{AnalyzerVersion: "version", Comments: comments}, nil
}

func (*analyzer) NotifyPushEvent(context.Context, *pb.PushEvent) (*pb.EventResponse, error) {
	return &pb.EventResponse{}, nil
}

func main() {
	listener, _ := pb.Listen("ipv4://0.0.0.0:9930")
	server := grpc.NewServer()
	pb.RegisterAnalyzerServer(server, &analyzer{})
	server.Serve(listener)
}
```


## Python

Install and use [`lookout_sdk`](https://pypi.org/project/lookout-sdk) python library:

```shell
$ pip install lookout-sdk
```

The analyzer class will extend [AnalyzerServicer](https://github.com/src-d/lookout-sdk/blob/master/python/lookout/sdk/service_analyzer_pb2_grpc.py#L34):

Once you register the analyzer and the [gRPC server](https://grpc.io/docs/tutorials/basic/python.html#starting-the-server) is runing, it will listen for requests from `lookoutd`.

_example:_

```python
#!/usr/bin/env python3
import grpc
from concurrent import futures
from lookout.sdk import pb

class Analyzer(pb.AnalyzerServicer):
    def NotifyReviewEvent(self, request, context):
        comments = []

        # Logic to generate 'comments' given the passed 'request'

        return pb.EventResponse(analyzer_version="version", comments=comments)

    def NotifyPushEvent(self, request, context):
        pass

server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
pb.add_analyzer_to_server(Analyzer(), server)
server.add_insecure_port("0.0.0.0:2021")
server.start()
time.sleep(60)
server.stop(0)
```
