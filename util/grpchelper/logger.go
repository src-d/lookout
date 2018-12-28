package grpchelper

import (
	"context"
	"path"
	"time"

	"github.com/src-d/lookout/util/ctxlog"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging"
	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
)

func getLogFn(l log.Logger, asDebug bool) func(msg string, args ...interface{}) {
	if asDebug {
		return l.Debugf
	}

	return l.Infof
}

// LogUnaryServerInterceptor returns a new unary server interceptor that logs
// request/response.
func LogUnaryServerInterceptor(asDebug bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		l := newServerRequestLogger(ctx, info.FullMethod)
		getLogFn(l, asDebug)("gRPC unary server call started")

		resp, err := handler(ctx, req)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("gRPC unary server call finished")

		return resp, err
	}
}

// LogStreamServerInterceptor returns a new streaming server interceptor that
// logs request/response.
func LogStreamServerInterceptor(asDebug bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		l := newServerRequestLogger(stream.Context(), info.FullMethod)
		getLogFn(l, asDebug)("gRPC streaming server call started")

		err := handler(srv, stream)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("gRPC streaming server call finished")

		return err
	}
}

// LogUnaryClientInterceptor returns a new unary client interceptor that logs
// the execution of external gRPC calls.
func LogUnaryClientInterceptor(asDebug bool) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		l := newClientRequestLogger(ctx, method)
		getLogFn(l, asDebug)("gRPC unary client call started")

		err := invoker(ctx, method, req, reply, cc, opts...)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("gRPC unary client call finished")

		return err
	}
}

// LogStreamClientInterceptor returns a new streaming client interceptor that
// logs the execution of external gRPC calls.
func LogStreamClientInterceptor(asDebug bool) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		startTime := time.Now()

		l := newClientRequestLogger(ctx, method)
		getLogFn(l, asDebug)("gRPC streaming client call started")

		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("gRPC streaming client call finished")

		return clientStream, err
	}
}

func newServerRequestLogger(ctx context.Context, fullMethod string) log.Logger {
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)

	_, logger := ctxlog.WithLogFields(ctx, log.Fields{
		"system":       "grpc",
		"span.kind":    "server",
		"grpc.service": service,
		"grpc.method":  method,
	})
	return logger
}

func newClientRequestLogger(ctx context.Context, fullMethod string) log.Logger {
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)

	_, logger := ctxlog.WithLogFields(ctx, log.Fields{
		"system":       "grpc",
		"span.kind":    "client",
		"grpc.service": service,
		"grpc.method":  method,
	})
	return logger
}

func newResponseLogger(l log.Logger, startTime time.Time, err error) log.Logger {
	fields := log.Fields{
		"grpc.start_time": startTime.Format(time.RFC3339),
		"grpc.code":       grpc_logging.DefaultErrorToCode(err),
		"duration":        time.Now().Sub(startTime),
	}

	if err != nil {
		fields["error"] = err
	}

	return l.With(fields)
}
