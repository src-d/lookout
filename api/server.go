package api

import (
	"fmt"
)

type DataReader interface {
	GetChanges(*ChangesRequest) ChangeScanner
}

type ChangeScanner interface {
	Next() bool
	Err() error
	Change() *ChangesResponse
	Close() error
}

type Server struct {
	DataReader DataReader
}

func NewServer(r DataReader) *Server {
	return &Server{
		DataReader: r,
	}
}

var _ DataServer = &Server{}

func (s *Server) GetChanges(req *ChangesRequest, srv Data_GetChangesServer) error {
	ctx := srv.Context()
	cancel := ctx.Done()
	iter := s.DataReader.GetChanges(req)
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
		if err := srv.Send(&ChangesResponse{
			Error: err.Error(),
		}); err != nil {
			_ = iter.Close()
			return err
		}
	}

	return iter.Close()
}
