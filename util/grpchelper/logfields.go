package grpchelper

import (
	"context"
	"encoding/json"

	"github.com/src-d/lookout/util/ctxlog"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	log "gopkg.in/src-d/go-log.v1"
)

const logFieldsKey = "log-fields"

// CtxlogUnaryClientInterceptor is a unary client interceptor that adds the
// ctxlog log.Fields to the grpc metadata, with the key 'logFieldsKey'.
func CtxlogUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	ctx = setLogFieldsMetadata(ctx)
	return invoker(ctx, method, req, reply, cc, opts...)
}

// CtxlogStreamClientInterceptor is a streaming client interceptor that adds the
// ctxlog log.Fields to the grpc metadata, with the key 'logFieldsKey'.
func CtxlogStreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	ctx = setLogFieldsMetadata(ctx)
	return streamer(ctx, desc, cc, method, opts...)
}

// setLogFieldsMetadata returns a new context with the ctxlog log.Fields stored
// into the grpc metadata, with the key 'logFieldsKey'.
func setLogFieldsMetadata(ctx context.Context) context.Context {
	f := ctxlog.Fields(ctx)

	// Delete the fields that should not cross to the gRPC server logs
	delete(f, "app")

	bytes, err := json.Marshal(f)
	if err != nil {
		ctxlog.Get(ctx).Errorf(err, "log.Fields could not be marshaled to JSON")
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, logFieldsKey, string(bytes))
}

// CtxlogUnaryServerInterceptor is a unary server interceptor that adds
// to the context a ctxlog configured with the log Fields found in the request
// metadata.
func CtxlogUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx = setContextLogger(ctx)
	return handler(ctx, req)
}

// CtxlogStreamServerInterceptor is a streaming server interceptor that
// adds to the context a ctxlog configured with the log Fields found in the
// request metadata.
func CtxlogStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = setContextLogger(stream.Context())

	return handler(srv, wrapped)
}

// setContextLogger returns a new context containing a ctxlog configured with
// the log Fields found in the given ctx metadata.
func setContextLogger(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md[logFieldsKey]) == 0 {
		return ctx
	}

	var f log.Fields
	err := json.Unmarshal([]byte(md[logFieldsKey][0]), &f)
	if err != nil {
		ctxlog.Get(ctx).Errorf(err, "log.Fields could not be unmarshaled from JSON")
		return ctx
	}

	// Delete the fields that we don't want overwritten by a gRPC client
	delete(f, "app")

	newCtx, _ := ctxlog.WithLogFields(ctx, f)
	return newCtx
}
