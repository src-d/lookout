package grpchelper

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	log "gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

// LogAsDebug allows to log gRPC messages as debug level instead of info level
var LogAsDebug = false

// NewServer creates new grpc.Server with custom message size
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	opts = append(opts,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			CtxlogStreamServerInterceptor,
			LogStreamServerInterceptor(LogAsDebug),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			CtxlogUnaryServerInterceptor,
			LogUnaryServerInterceptor(LogAsDebug),
		)),
	)

	return pb.NewServer(opts...)
}

// DialContext creates a client connection to the given target with custom message size
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts,
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			LogStreamClientInterceptor(LogAsDebug),
			CtxlogStreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			LogUnaryClientInterceptor(LogAsDebug),
			CtxlogUnaryClientInterceptor,
		)),
	)

	return pb.DialContext(ctx, target, opts...)
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
