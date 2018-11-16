# Whan an Analyzer Is?

_For detailed information about the different parts of **source{d} Lookout**, and how they interact you can go to the [**source{d} Lookout Architecture** guide](architecture.md)._

_For information about how to create your own Analyzer, go to the [**source{d} Lookout Analyzer Creation** guide](analyzers-creation.md)._

Essentially, an analyzer is just a [gRPC server](https://grpc.io/docs/guides/#overview) that will be called by **source{d} Lookout** using [protocol buffers](https://developers.google.com/protocol-buffers) whenever a Pull Request is ready to be analyzed, or it is updated.

To do so, the analyzer must implement [Analyzer service](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_analyzer.proto#L30) interface:

```protobuf
service Analyzer {
  rpc NotifyReviewEvent (ReviewEvent) returns (EventResponse);
  rpc NotifyPushEvent (PushEvent) returns (EventResponse);
}
```

To process the Pull Request, the analyzer can request a stream ([go](https://grpc.io/docs/tutorials/basic/go.html#server-side-streaming-rpc-1), [python](https://grpc.io/docs/tutorials/basic/python.html#response-streaming-rpc)) of files and changes from [**Lookout DataService**](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_data.proto#L27) that **Lookout** exposes, by default, on `localhost:10301`.


## NotifyReviewEvent

The main responsibility of the Analyzer will be the `NotifyReviewEvent` procedure, called from **Lookout** server when a Pull Requests should be reviewed.

The `ReviewEvent` will cause the analysis of the Pull Request by the Analyzer; the result of the analysis will be an `EventResponse` containing the `Comments` to be posted to GitHub.


## NotifyPushEvent

_**Important note**: The response for `NotifyPushEvent` allows a series of `Comments`, but this is a placeholder for future functionality. The Analyzer is not expected to return any comment in the current version._

The `NotifyPushEvent` procedure is called from **Lookout** server when there are new commits pushed to any watched repository.

The Analyzer is not enforced to do anything with this notification. It could be used, for example, to re-train an internal model using the new contents of the master branch.
