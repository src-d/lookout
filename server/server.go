package server

import (
	"fmt"

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
