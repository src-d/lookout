# Implementing Your Own Analyzer

_For a brief description about what an analyzer is, you can read [**source{d} lookout Analyzers** documentation](analyzers.md)_

_Please refer to the [**official Protocol Buffers** documentation](https://developers.google.com/protocol-buffers/) to learn how to get started with Protocol Buffers._

To implement your own analyzer you only need to create a gRPC service implementing [Analyzer service](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_analyzer.proto#L30) interface:

```protobuf
service Analyzer {
  rpc NotifyReviewEvent (ReviewEvent) returns (EventResponse);
  rpc NotifyPushEvent (PushEvent) returns (EventResponse);
}
```

You can create a new analyzer in any language that supports protocol buffers, generating code from [the `.proto` definitions](https://github.com/src-d/lookout-sdk/tree/master/proto/lookout/sdk). The resulting code will provide data access classes, with accessors for each field, as well as methods to serialize/parse the message structures to/from bytes.

## Caveats

All the analyzers should consider [the caveats described by the SDK](https://github.com/src-d/lookout-sdk#caveats).


## Fetching Changes, UASTs or Languages from lookout DataService

The analyzer should not need to take care of managing git data, nor storing the repositories, nor guessing the programming language used in the files nor parsing its UASTs; to do so, it should request **lookout DataService** for such data.

You can read more about it in [**lookout DataService** section](architecture.md#lookout-dataservice).


## How to Locally Try an Analyzer

_Please refer to [**lookout-sdk** docs](lookout-sdk.md) to see how to locally test an analyzer without accessing GitHub at all._


# Using Pregenerated Code from the SDK

If you're creating the analyzer in Golang or Python, you'll find pre-generated libraries in the [lookout-sdk repository](https://github.com/src-d/lookout-sdk) so you'll have to generate nothing.

You can do as it follows:

## Golang

Import and use `github.com/src-d/lookout-sdk/pb`.

The analyzer must implement the [AnalyzerClient interface](https://github.com/src-d/lookout-sdk/blob/master/pb/service_analyzer.pb.go#L129):

```go
import github.com/src-d/lookout-sdk/pb
type AnalyzerClient interface {
  NotifyReviewEvent(context.Context, *pb.ReviewEvent, opts ...grpc.CallOption) (*pb.EventResponse, error)
  NotifyPushEvent(context.Context, *pb.PushEvent, opts ...grpc.CallOption) (*pb.EventResponse, error)
}
```

Once the gRPC server is run, it will listen for requests from `lookoutd`.


## Python

Install and use [`lookout_sdk`](https://pypi.org/project/lookout-sdk) python library:

```shell
$ pip install lookout-sdk
```

The analyzer class will extend [AnalyzerServicer](https://github.com/src-d/lookout-sdk/blob/master/python/lookout/sdk/service_analyzer_pb2_grpc.py#34):

```python
class AnalyzerServicer(object):
  def NotifyReviewEvent(self, request, context):
  def NotifyPushEvent(self, request, context):
```

Start the [gRPC server](https://grpc.io/docs/tutorials/basic/python.html#starting-the-server) and add an Analyzer instance to it. Once you do it, the analyzer will listen for requests from `lookoutd`.
