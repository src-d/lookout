package grpchelper

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"google.golang.org/grpc"
	log "gopkg.in/src-d/go-log.v1"
)

var maxMessageSize = 100 * 1024 * 1024 // 100mb

// SetMaxMessageSize overrides default grpc max. message size to send/receive to/from clients
func SetMaxMessageSize(size int) {
	if size >= 2048 {
		// Setting the hard limit of message size to less than 2GB since
		// it may overflow an int value, and it should be big enough
		log.Errorf(fmt.Errorf("max-message-size too big (limit is 2047MB): %d", size), "SetMaxMessageSize")
		os.Exit(1)
	}

	maxMessageSize = size * 1024 * 1024
}

//TODO: https://github.com/grpc/grpc-go/issues/1911

// ToNetListenerAddress converts a gRPC URL to a network+address consumable by
// net.Listen. For example:
//   ipv4://127.0.0.1:8080 -> (tcp4, 127.0.0.1:8080)
func ToNetListenerAddress(target string) (network, address string, err error) {
	u, err := url.Parse(target)
	if err != nil {
		return
	}

	if u.Scheme == "dns" {
		err = fmt.Errorf("dns:// not supported")
		return
	}

	if u.Scheme == "unix" {
		network = "unix"
		address = u.Path
		return
	}

	address = u.Host
	switch u.Scheme {
	case "ipv4":
		network = "tcp4"
	case "ipv6":
		network = "tcp6"
	default:
		err = fmt.Errorf("scheme not supported: %s", u.Scheme)
	}

	return
}

// ToGoGrpcAddress converts a standard gRPC target name to a
// one that is supported by grpc-go.
func ToGoGrpcAddress(address string) (string, error) {
	n, a, err := ToNetListenerAddress(address)
	if err != nil {
		return "", err
	}

	if n == "unix" {
		return fmt.Sprintf("unix:%s", a), nil
	}

	return a, nil
}

// Listen is equivalent to standard net.Listen, but taking gRPC URL as input.
func Listen(address string) (net.Listener, error) {
	n, a, err := ToNetListenerAddress(address)
	if err != nil {
		return nil, err
	}

	return net.Listen(n, a)
}

// NewGrpcServer creates new grpc.Server with custom message size
func NewServer(opts ...grpc.ServerOption) *grpc.Server {
	opts = append(opts, grpc.MaxRecvMsgSize(maxMessageSize), grpc.MaxSendMsgSize(maxMessageSize))

	return grpc.NewServer(opts...)
}

// GrpcDialContext creates a client connection to the given target with custom message size
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(maxMessageSize),
		grpc.MaxCallSendMsgSize(maxMessageSize),
	))

	return grpc.DialContext(ctx, target, opts...)
}
