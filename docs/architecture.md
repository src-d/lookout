# source{d} lookout Architecture

_You will find the project background, goals, constraints, solution details and other info in [**source{d} lookout DesignDoc**](https://docs.google.com/document/d/1pqz-_SHO5BsJE-aa8o_bAY3r5vR67amnWN8-qZc2UgY/edit#)_

![lookout service sequence diagram](assets/lookout-seq-diagram.png)

You can [edit this image](https://mermaidjs.github.io/mermaid-live-editor/#/edit/eyJjb2RlIjoic2VxdWVuY2VEaWFncmFtXG4gICAgcGFydGljaXBhbnQgR2l0SHViXG4gICAgcGFydGljaXBhbnQgU2VydmVyXG4gICAgcGFydGljaXBhbnQgRGF0YVNlcnZpY2VcbiAgICBwYXJ0aWNpcGFudCBCYmxmc2hcbiAgICBwYXJ0aWNpcGFudCBBbmFseXplclxuICAgIGxvb3AgSGVhbHRoY2hlY2tcbiAgICAgICAgU2VydmVyIC0-IFNlcnZlcjoga2VlcCB3YWl0aW5nXG4gICAgZW5kXG4gICAgbG9vcCByZWdpc3RlcmVkIFJlcG9zXG4gICAgICAgIFNlcnZlciAtPj4gR2l0SHViOiBQb2xsOiBQUnMgZXZlbnRzXG4gICAgICAgIGxvb3AgcmVnaXN0ZXJlZCBBbmFseXplcnNcbiAgICAgICAgICAgIFNlcnZlciAtPj4gK0FuYWx5emVyOiAoZ1JQQykgTm90aWZ5UmV2aWV3RXZlbnRcbiAgICAgICAgICAgIEFuYWx5emVyIC0-PiArRGF0YVNlcnZpY2U6IChnUlBDKSBHZXRDaGFuZ2VzIC8gR2V0RmlsZXNcbiAgICAgICAgICAgIG9wdCByZXEuV2FudFVBU1Q_XG4gICAgICAgICAgICAgICAgRGF0YVNlcnZpY2UgLT4-ICtCYmxmc2g6IChnUlBDKSBQYXJzZVxuICAgICAgICAgICAgICAgIEJibGZzaCAtLT4-IC1EYXRhU2VydmljZTogdWFzdC5Ob2RlXG4gICAgICAgICAgICBlbmRcbiAgICAgICAgICAgIERhdGFTZXJ2aWNlIC0tPj4gLUFuYWx5emVyOiBzdHJlYW0gb2YgQ2hhbmdlIC8gRmlsZVxuICAgICAgICAgICAgQW5hbHl6ZXIgLS0-PiAtU2VydmVyOiBDb21tZW50c1xuICAgICAgICBlbmRcbiAgICBTZXJ2ZXIgLT4-IEdpdEh1YjogUG9zdCBhbGwgY29tbWVudHNcbiAgICBlbmRcbiIsIm1lcm1haWQiOnsidGhlbWUiOiJkZWZhdWx0In19) using [mermaid](https://mermaidjs.github.io). ([sourcecode](assets/lookout-seq-diagram.md))

Lookout consists of the following components:


## Server

It is the main component, running in a separate process.
It is responsible for orchestrating all the other services.
It takes review requests made by an external code review system, calls the registered analyzers to review the changes, and posts the results back.


## lookout DataService

_For the gRPC **lookout DataService** service definiton you can take al look to **[`service_data.proto`](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_data.proto#L27)**_

**lookout DataService** deals with actual Git repositories; it is responsible for fetching and storing git repositories.

**lookout DataService** is also exposed by **lookout** as a gRPC service &mdash;by default, on `localhost:10301`&mdash; that can be called by the analyzers to request a stream (ie. [go](https://grpc.io/docs/tutorials/basic/go.html#server-side-streaming-rpc-1), [python](https://grpc.io/docs/tutorials/basic/python.html#response-streaming-rpc)) of files and changes from **lookout DataService** that **lookout** exposes.

The [ChangesRequest](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_data.proto#L58) or [FilesRequest](https://github.com/src-d/lookout-sdk/blob/master/proto/lookout/sdk/service_data.proto#L69) can be configured the to ask either for all files, or just the changed ones, as well as UASTs, language, full file content and/or exclude some paths: by regexp, or just all [vendored paths](https://github.com/github/linguist/blob/master/lib/linguist/vendor.yml).


## Analyzer

_Find more info about what an analyzer is and how to develop your own analyzer in the [**source{d} lookout Analyzers** documentation](analyzers.md)_

An analyzer is a gRPC service that will be called by the [Server](#server) to perform the smart code analysis, and it will return a set of `Comments` as the result of the analysis.

They are not part of **lookout** repository so they can be developed by third parties.

Lookout Server will call all the Analyzers that were already registered when it was started.


# SDK

The [lookout-sdk](https://github.com/src-d/lookout-sdk) repository is a toolkit for writing new analyzers. It contains:
- `.proto` interface definitions for all **lookout** gRPC services.
- The pre-generated code in Go and Python that provides an easy access to the **lookout DataService** gRPC service; and low-level helpers to workaround some protobuf/gRPC caveats.
- Two simple quickstart examples.
