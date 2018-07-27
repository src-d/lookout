package grpchelper

import (
	"context"
	"path"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
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

// UnaryServerInterceptor returns a new unary server interceptors that logs request/response.
func UnaryServerInterceptor(l log.Logger, asDebug bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		l := newServerRequestLogger(l, info.FullMethod)
		getLogFn(l, asDebug)("unary server call started")

		resp, err := handler(ctx, req)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("unary server call finished")

		return resp, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor that logs request/response.
func StreamServerInterceptor(l log.Logger, asDebug bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		l := newServerRequestLogger(l, info.FullMethod)
		getLogFn(l, asDebug)("streaming server call started")

		wrapped := grpc_middleware.WrapServerStream(stream)
		err := handler(srv, wrapped)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("streaming server call finished")

		return err
	}
}

// UnaryClientInterceptor returns a new unary client interceptor that logs the execution of external gRPC calls.
func UnaryClientInterceptor(l log.Logger, asDebug bool) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		startTime := time.Now()

		l := newClientRequestLogger(l, method)
		getLogFn(l, asDebug)("unary client call started")

		err := invoker(ctx, method, req, reply, cc, opts...)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("streaming client call finished")

		return err
	}
}

// StreamClientInterceptor returns a new striming client interceptor that logs the execution of external gRPC calls.
func StreamClientInterceptor(l log.Logger, asDebug bool) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		startTime := time.Now()

		l := newClientRequestLogger(l, method)
		getLogFn(l, asDebug)("streaming client call started")

		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		getLogFn(newResponseLogger(l, startTime, err), asDebug)("streaming client call finished")

		return clientStream, err
	}
}

func newServerRequestLogger(l log.Logger, fullMethod string) log.Logger {
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)

	return l.With(log.Fields{
		"system":       "grpc",
		"span.kind":    "server",
		"grpc.service": service,
		"grpc.method":  method,
	})
}

func newClientRequestLogger(l log.Logger, fullMethod string) log.Logger {
	service := path.Dir(fullMethod)[1:]
	method := path.Base(fullMethod)

	return l.With(log.Fields{
		"system":       "grpc",
		"span.kind":    "client",
		"grpc.service": service,
		"grpc.method":  method,
	})
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
