package server

import (
	"fmt"
	"net"
	"net/url"

	"github.com/src-d/lookout/api"
)

type Server struct {
	Service api.Service
}

func NewServer(s api.Service) *Server {
	return &Server{
		Service: s,
	}
}

var _ api.DataServer = &Server{}

func (s *Server) GetChanges(req *api.ChangesRequest,
	srv api.Data_GetChangesServer) error {

	ctx := srv.Context()
	cancel := ctx.Done()
	iter, err := s.Service.GetChanges(req)
	if err != nil {
		return err
	}

	for iter.Next() {
		select {
		case <-cancel:
			_ = iter.Close()
			return fmt.Errorf("request canceled: %s", ctx.Err())
		default:
		}

		if err := srv.Send(iter.Change()); err != nil {
			_ = iter.Close()
			return err
		}
	}

	if err := iter.Err(); err != nil {
		_ = iter.Close()
		return err
	}

	return iter.Close()
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
