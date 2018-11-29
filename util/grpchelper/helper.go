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

// NewServer creates new grpc.Server with custom message size
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	opts = append(opts,
		grpc.StreamInterceptor(StreamServerInterceptor(log.DefaultLogger, LogAsDebug)),
		grpc.UnaryInterceptor(UnaryServerInterceptor(log.DefaultLogger, LogAsDebug)),
	)

	return pb.NewServer(opts...)
}

// DialContext creates a client connection to the given target with custom message size
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts,
		grpc.WithStreamInterceptor(StreamClientInterceptor(log.DefaultLogger, LogAsDebug)),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor(log.DefaultLogger, LogAsDebug)),
	)

	return pb.DialContext(ctx, target, opts...)
}

// LogConnStatusChanges logs gRPC connection status changes
func LogConnStatusChanges(ctx context.Context, l log.Logger, conn *grpc.ClientConn) {
	state := conn.GetState()
	for {
		if conn.WaitForStateChange(ctx, state) {
			state = conn.GetState()
			if state == connectivity.TransientFailure {
				l.Warningf("connection failed")
			} else {
				l.Infof("connection state changed to '%s'", state)
			}
		} else {
			// ctx expired / canceled, stop listing
			return
		}
	}
}
