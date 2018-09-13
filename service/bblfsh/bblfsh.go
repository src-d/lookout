package bblfsh

import (
	"context"
	"strings"

	"github.com/src-d/lookout"
	"github.com/src-d/lookout/util/ctxlog"

	"google.golang.org/grpc"
	"gopkg.in/bblfsh/sdk.v1/protocol"
	"gopkg.in/bblfsh/sdk.v1/uast"
)

// Service implements data service interface which adds UAST to the responses
type Service struct {
	changes lookout.ChangeGetter
	files   lookout.FileGetter
	client  protocol.ProtocolServiceClient
}

var _ lookout.ChangeGetter = &Service{}
var _ lookout.FileGetter = &Service{}

// NewService creates new bblfsh Service
func NewService(changes lookout.ChangeGetter, files lookout.FileGetter, conn *grpc.ClientConn) *Service {
	return &Service{
		changes: changes,
		files:   files,
		client:  protocol.NewProtocolServiceClient(conn),
	}
}

// GetChanges returns a ChangeScanner that scans all changes according to the request.
func (s *Service) GetChanges(ctx context.Context, req *lookout.ChangesRequest) (lookout.ChangeScanner, error) {
	if req.WantUAST {
		req.WantContents = true
		req.WantLanguage = true
	}

	changes, err := s.changes.GetChanges(ctx, req)
	if err != nil {
		return nil, err
	}

	if !req.WantUAST {
		return changes, nil
	}

	return &ChangeScanner{
		underlying: changes,
		BaseScanner: BaseScanner{
			ctx:    ctx,
			client: s.client,
		},
	}, nil
}

// GetFiles returns a FilesScanner that scans all files according to the request.
func (s *Service) GetFiles(ctx context.Context, req *lookout.FilesRequest) (lookout.FileScanner, error) {
	if req.WantUAST {
		req.WantContents = true
		req.WantLanguage = true
	}

	files, err := s.files.GetFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	if !req.WantUAST {
		return files, nil
	}

	return &FileScanner{
		underlying: files,
		BaseScanner: BaseScanner{
			ctx:    ctx,
			client: s.client,
		},
	}, nil
}

type BaseScanner struct {
	ctx    context.Context
	client protocol.ProtocolServiceClient
	err    error
}

func (s *BaseScanner) processFile(f *lookout.File) error {
	if f == nil {
		return nil
	}

	ctxlog.Get(s.ctx).Debugf("parsing uast for file: %s", f.Path)

	var err error
	f.UAST, err = s.parseFile(f)
	if err != nil {
		return err
	}

	return nil
}

func (s *BaseScanner) parseFile(f *lookout.File) (*uast.Node, error) {
	if f.Path == "" {
		return nil, nil
	}

	req := &protocol.ParseRequest{
		Filename: f.Path,
		Content:  string(f.Content),
		Encoding: protocol.UTF8,
		Language: strings.ToLower(f.Language),
	}
	resp, err := s.client.Parse(s.ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Status != protocol.Ok {
		return nil, nil
	}

	return resp.UAST, nil
}

// ChangeScanner is a scanner for changes.
type ChangeScanner struct {
	BaseScanner
	underlying lookout.ChangeScanner
	val        *lookout.Change
}

func (s *ChangeScanner) Next() bool {
	if s.err != nil {
		return false
	}

	if !s.underlying.Next() {
		return false
	}

	s.val = s.underlying.Change()

	if err := s.processFile(s.val.Base); err != nil {
		s.err = err
		return false
	}

	if err := s.processFile(s.val.Head); err != nil {
		s.err = err
		return false
	}

	return true
}

func (s *ChangeScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.underlying.Err()
}

func (s *ChangeScanner) Change() *lookout.Change {
	return s.val
}

func (s *ChangeScanner) Close() error {
	return s.underlying.Close()
}

// FileScanner is a scanner for files.
type FileScanner struct {
	BaseScanner
	underlying lookout.FileScanner
	val        *lookout.File
}

func (s *FileScanner) Next() bool {
	if s.err != nil {
		return false
	}

	if !s.underlying.Next() {
		return false
	}

	s.val = s.underlying.File()

	if err := s.processFile(s.val); err != nil {
		s.err = err
		return false
	}

	return true
}

func (s *FileScanner) Err() error {
	if s.err != nil {
		return s.err
	}

	return s.underlying.Err()
}

func (s *FileScanner) File() *lookout.File {
	return s.val
}

func (s *FileScanner) Close() error {
	return s.underlying.Close()
}
