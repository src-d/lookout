package lookout

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
	"gopkg.in/src-d/lookout-sdk.v0/pb"
)

type ChangesRequest = pb.ChangesRequest
type FilesRequest = pb.FilesRequest
type Change = pb.Change
type File = pb.File

// ChangeGetter is used to retrieve code changes.
type ChangeGetter interface {
	// GetChanges returns a ChangeScanner that scans all changes according
	// to the request.
	GetChanges(context.Context, *ChangesRequest) (ChangeScanner, error)
}

// FileGetter is used to retrieve all code for a revision.
type FileGetter interface {
	// GetFiles returns a FilesScanner that scans all files according
	// to the request.
	GetFiles(context.Context, *FilesRequest) (FileScanner, error)
}

func RegisterDataServer(s *grpc.Server, srv *DataServerHandler) {
	pb.RegisterDataServer(s, srv)
}

// ChangeScanner is a scanner for changes.
type ChangeScanner interface {
	// Next advances the scanner to the next change. It returns true if a new
	// change is found, and false otherwise. After the user is done scanning,
	// Err must be called to check if all changes were consumed or there was an
	// error.
	Next() bool
	// Err returns any error found during scanning.
	Err() error
	// Change returns the current change.
	Change() *Change
	// Close closes the scanner.
	Close() error
}

// FileScanner is a scanner for files.
type FileScanner interface {
	// Next advances the scanner to the next file. It returns true if a new
	// file is found, and false otherwise. After the user is done scanning,
	// Err must be called to check if all files were consumed or there was an
	// error.
	Next() bool
	// Err returns any error found during scanning.
	Err() error
	// File returns the current file.
	File() *File
	// Close closes the scanner.
	Close() error
}

type DataServerHandler struct {
	ChangeGetter ChangeGetter
	FileGetter   FileGetter
}

var _ pb.DataServer = &DataServerHandler{}

func (s *DataServerHandler) GetChanges(req *ChangesRequest,
	srv pb.Data_GetChangesServer) (err error) {

	ctx := srv.Context()
	cancel := ctx.Done()
	iter, err := s.ChangeGetter.GetChanges(ctx, req)
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(iter, &err)

	for iter.Next() {
		select {
		case <-cancel:
			return fmt.Errorf("request canceled: %s", ctx.Err())
		default:
		}

		if err := srv.Send(iter.Change()); err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return err
}

func (s *DataServerHandler) GetFiles(req *FilesRequest, srv pb.Data_GetFilesServer) (err error) {
	ctx := srv.Context()
	cancel := ctx.Done()
	iter, err := s.FileGetter.GetFiles(ctx, req)
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(iter, &err)

	for iter.Next() {
		select {
		case <-cancel:
			return fmt.Errorf("request canceled: %s", ctx.Err())
		default:
		}

		if err := srv.Send(iter.File()); err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return err
}

type DataClient struct {
	dataClient pb.DataClient
}

func NewDataClient(cc *grpc.ClientConn) *DataClient {
	return &DataClient{
		dataClient: pb.NewDataClient(cc),
	}
}

func (c *DataClient) GetChanges(ctx context.Context, in *ChangesRequest, opts ...grpc.CallOption) (
	ChangeScanner, error) {

	resp, err := c.dataClient.GetChanges(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	if err := resp.CloseSend(); err != nil {
		return nil, err
	}

	return &ClientChangeScanner{
		client: resp,
	}, nil

}

func (c *DataClient) GetFiles(ctx context.Context, in *FilesRequest, opts ...grpc.CallOption) (
	FileScanner, error) {

	resp, err := c.dataClient.GetFiles(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	if err := resp.CloseSend(); err != nil {
		return nil, err
	}

	return &ClientFileScanner{
		client: resp,
	}, nil

}

type ClientChangeScanner struct {
	client pb.Data_GetChangesClient
	val    *Change
	err    error
	done   bool
}

func (s *ClientChangeScanner) Next() bool {
	if s.done {
		return false
	}

	s.val, s.err = s.client.Recv()
	if s.err == io.EOF {
		s.err = nil
		s.done = true
		return false
	}

	if s.err != nil {
		s.done = true
		return false
	}

	return true
}

func (s *ClientChangeScanner) Err() error {
	return s.err
}

func (s *ClientChangeScanner) Change() *Change {
	return s.val
}

func (s *ClientChangeScanner) Close() error {
	return nil
}

type ClientFileScanner struct {
	client pb.Data_GetFilesClient
	val    *File
	err    error
	done   bool
}

func (s *ClientFileScanner) Next() bool {
	if s.done {
		return false
	}

	s.val, s.err = s.client.Recv()
	if s.err == io.EOF {
		s.err = nil
		s.done = true
		return false
	}

	if s.err != nil {
		s.done = true
		return false
	}

	return true
}

func (s *ClientFileScanner) Err() error {
	return s.err
}

func (s *ClientFileScanner) File() *File {
	return s.val
}

func (s *ClientFileScanner) Close() error {
	return nil
}

// FnChangeScanner implements ChangeScanner using functions
type FnChangeScanner struct {
	Scanner ChangeScanner
	Fn      func(*Change) (bool, error)
	OnStart func() error
	val     *Change
	started bool
	done    bool
	err     error
}

func (s *FnChangeScanner) Next() bool {
	if s.done {
		return false
	}

	if !s.started {
		defer func() { s.started = true }()

		if s.OnStart != nil {
			s.err = s.OnStart()
			if s.err != nil {
				s.done = true
				return false
			}
		}
	}

	for s.Scanner.Next() {
		ch := s.Scanner.Change()
		skip, err := s.Fn(ch)
		if err != nil {
			s.done = true
			s.err = err
			return false
		}
		if skip {
			continue
		}

		s.val = ch
		return true
	}

	s.done = true
	return false
}

func (s *FnChangeScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *FnChangeScanner) Change() *Change {
	return s.val
}

func (s *FnChangeScanner) Close() error {
	return s.Scanner.Close()
}

// FnFileScanner implements FileScanner using functions
type FnFileScanner struct {
	Scanner FileScanner
	Fn      func(*File) (bool, error)
	OnStart func() error
	val     *File
	started bool
	done    bool
	err     error
}

func (s *FnFileScanner) Next() bool {
	if s.done {
		return false
	}

	if !s.started {
		defer func() { s.started = true }()

		if s.OnStart != nil {
			s.err = s.OnStart()
			if s.err != nil {
				s.done = true
				return false
			}
		}
	}

	for s.Scanner.Next() {
		f := s.Scanner.File()
		skip, err := s.Fn(f)
		if err != nil {
			s.done = true
			s.err = err
			return false
		}
		if skip {
			continue
		}

		s.val = f
		return true
	}

	s.done = true
	return false
}

func (s *FnFileScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.Scanner.Err()
}

func (s *FnFileScanner) File() *File {
	return s.val
}

func (s *FnFileScanner) Close() error {
	return s.Scanner.Close()
}
