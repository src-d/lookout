package pb

import (
	"context"
	"encoding/json"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const logFieldsKeyMeta = "log-fields"

// CtxlogUnaryClientInterceptor is an unary client interceptor that adds
// the log fields to the grpc metadata, with the key 'logFieldsKeyMeta'.
func CtxlogUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return invoker(setLogFieldsMetadata(ctx), method, req, reply, cc, opts...)
}

// CtxlogStreamClientInterceptor is a streaming client interceptor that adds
// the log fields to the grpc metadata, with the key 'logFieldsKeyMeta'.
func CtxlogStreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return streamer(setLogFieldsMetadata(ctx), desc, cc, method, opts...)
}

// setLogFieldsMetadata returns a new context with the log fields stored
// into the grpc metadata, with the key 'logFieldsKeyMeta'.
func setLogFieldsMetadata(ctx context.Context) context.Context {
	bytes, _ := json.Marshal(GetLogFields(ctx))
	return metadata.AppendToOutgoingContext(ctx, logFieldsKeyMeta, string(bytes))
}

// CtxlogUnaryServerInterceptor is an unary server interceptor that adds
// to the context the log fields found in the request metadata with the key `logFieldsKeyMeta`.
func CtxlogUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx = setContextLogger(ctx)
	return handler(ctx, req)
}

// CtxlogStreamServerInterceptor is a streaming server interceptor that adds
// to the context the log fields found in the request metadata with the key `logFieldsKeyMeta`.
func CtxlogStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = setContextLogger(stream.Context())

	return handler(srv, wrapped)
}

// setContextLogger returns a new context containing with the log fields found
// in the given request metadata with the key `logFieldsKeyMeta`.
func setContextLogger(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md[logFieldsKeyMeta]) == 0 {
		return ctx
	}

	var fields Fields
	if err := json.Unmarshal([]byte(md[logFieldsKeyMeta][0]), &fields); err != nil {
		return ctx
	}

	return AddLogFields(ctx, fields)
}
