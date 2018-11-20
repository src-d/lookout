package grpchelper

import (
	"context"
	"strings"

	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// NewBblfshProxyServer creates gRPC server that proxies call to bblfsh
func NewBblfshProxyServer(addr string) (*grpc.Server, error) {
	// bblfsh connection for proxy
	bblfshConn, err := grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithInsecure(), grpc.WithCodec(proxy.Codec()),
	)
	if err != nil {
		return nil, err
	}

	// bblfsh proxy director
	director := func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		md, ok := metadata.FromIncomingContext(ctx)

		if ok && strings.HasPrefix(fullMethodName, "/gopkg.in.bblfsh.") {
			outCtx := metadata.NewOutgoingContext(ctx, md.Copy())
			return outCtx, bblfshConn, nil
		}

		return nil, nil, grpc.Errorf(codes.Unimplemented, "Unknown method")
	}

	return NewServer(
		grpc.CustomCodec(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(director)),
	), nil
}
