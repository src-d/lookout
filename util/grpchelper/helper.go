package grpchelper

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

// LogAsDebug allows to log gRPC messages as debug level instead of info level
var LogAsDebug = false

var logFn = func(fields pb.Fields, format string, args ...interface{}) {
	l := log.With(log.Fields(fields))
	if LogAsDebug {
		l.Debugf(format, args...)
	} else {
		l.Infof(format, args...)
	}
}

// NewServer creates new grpc.Server with custom options and log interceptors
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	return pb.NewServerWithInterceptors(
		[]grpc.StreamServerInterceptor{
			pb.LogStreamServerInterceptor(logFn),
			CtxlogStreamServerInterceptor,
		},
		[]grpc.UnaryServerInterceptor{
			pb.LogUnaryServerInterceptor(logFn),
			CtxlogUnaryServerInterceptor,
		},
		opts...,
	)
}

// DialContext creates a client connection to the given target with custom
// options and log interceptors
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return pb.DialContextWithInterceptors(
		ctx, target,
		[]grpc.StreamClientInterceptor{
			CtxlogStreamClientInterceptor,
			pb.LogStreamClientInterceptor(logFn),
		},
		[]grpc.UnaryClientInterceptor{
			CtxlogUnaryClientInterceptor,
			pb.LogUnaryClientInterceptor(logFn),
		},
		opts...,
	)
}

func logConnStatus(l log.Logger, state connectivity.State) {
	if state == connectivity.TransientFailure {
		l.Warningf("connection failed")
	} else {
		l.Infof("connection state changed to '%s'", state)
	}
}

// LogConnStatusChanges logs gRPC connection status changes
func LogConnStatusChanges(ctx context.Context, l log.Logger, conn *grpc.ClientConn) {
	state := conn.GetState()
	logConnStatus(l, state)

	for {
		if conn.WaitForStateChange(ctx, state) {
			state = conn.GetState()
			logConnStatus(l, state)
		} else {
			// ctx expired / canceled, stop listing
			return
		}
	}
}
