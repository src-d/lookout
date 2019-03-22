package pb

import (
	"context"
	"path"
	"time"

	"google.golang.org/grpc"
)

// LogFn is the function used to log the messages
type LogFn func(fields Fields, format string, args ...interface{})

// LogUnaryServerInterceptor returns a new unary server interceptor that logs
// request/response.
func LogUnaryServerInterceptor(log LogFn) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		logCtx := buildServerRequestCtx(ctx, info.FullMethod)
		log(GetLogFields(logCtx), "gRPC unary server call started")

		resp, err := handler(ctx, req)

		logCtx = buildResponseLoggerCtx(logCtx, startTime, err)
		log(GetLogFields(logCtx), "gRPC unary server call finished")

		return resp, err
	}
}

// LogStreamServerInterceptor returns a new streaming server interceptor that
// logs request/response.
func LogStreamServerInterceptor(log LogFn) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		logCtx := buildServerRequestCtx(stream.Context(), info.FullMethod)
		log(GetLogFields(logCtx), "gRPC streaming server call started")

		err := handler(srv, stream)

		logCtx = buildResponseLoggerCtx(logCtx, startTime, err)
		log(GetLogFields(logCtx), "gRPC streaming server call finished")

		return err
	}
}

// LogUnaryClientInterceptor returns a new unary client interceptor that logs
// the execution of external gRPC calls.
func LogUnaryClientInterceptor(log LogFn) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		logCtx := buildClientRequestCtx(ctx, method)
		log(GetLogFields(logCtx), "gRPC unary client call started")

		err := invoker(ctx, method, req, reply, cc, opts...)

		logCtx = buildResponseLoggerCtx(logCtx, startTime, err)
		log(GetLogFields(logCtx), "gRPC unary client call finished")

		return err
	}
}

// LogStreamClientInterceptor returns a new streaming client interceptor that
// logs the execution of external gRPC calls.
func LogStreamClientInterceptor(log LogFn) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		startTime := time.Now()

		logCtx := buildClientRequestCtx(ctx, method)
		log(GetLogFields(logCtx), "gRPC streaming client call started")

		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		logCtx = buildResponseLoggerCtx(ctx, startTime, err)
		log(GetLogFields(logCtx), "gRPC streaming client call finished")

		return clientStream, err
	}
}

func buildServerRequestCtx(ctx context.Context, fullMethod string) context.Context {
	return buildRequestCtx(ctx, "server", fullMethod)
}

func buildClientRequestCtx(ctx context.Context, fullMethod string) context.Context {
	return buildRequestCtx(ctx, "client", fullMethod)
}

func buildRequestCtx(ctx context.Context, kind, fullMethod string) context.Context {
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)

	return AddLogFields(ctx, Fields{
		"system":       "grpc",
		"span.kind":    kind,
		"grpc.service": service,
		"grpc.method":  method,
	})
}

func buildResponseLoggerCtx(ctx context.Context, startTime time.Time, err error) context.Context {
	fields := Fields{
		"grpc.start_time": startTime.Format(time.RFC3339),
		"grpc.code":       grpc.Code(err),
		"duration":        time.Now().Sub(startTime),
	}

	if err != nil {
		fields["error"] = err
	}

	return AddLogFields(ctx, fields)
}
